package golang

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"strings"
	"text/template"
	"unicode"

	"github.com/pgavlin/warp/exec"
	"github.com/pgavlin/warp/wasm"
	"github.com/pgavlin/warp/wasm/code"
	"github.com/pgavlin/warp/wasm/validate"
)

// ErrInvalidMemoryIndex indicates that the memory index associated with a data section is
// not valid.
var ErrInvalidMemoryIndex = fmt.Errorf("invalid memory index")

type moduleCompiler struct {
	isCommand         bool
	noInternalThreads bool
	useRawPointers    bool

	packageName  string
	name         string
	exportedName string
	module       *wasm.Module

	importedFunctions []wasm.FunctionSig
	importedMemory    *wasm.ImportEntry
	importedTable     *wasm.ImportEntry
	importedGlobals   []wasm.GlobalVar

	exportedGlobals   map[uint32]bool
	exportedFunctions map[uint32]bool

	functionNames map[uint32]string
	functions     []functionCompiler
}

func unexportName(name string) string {
	runes := []rune(name)
	if !unicode.IsUpper(runes[0]) {
		return name
	}
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func exportName(name string) string {
	runes := []rune(name)
	if unicode.IsUpper(runes[0]) {
		return name
	}
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func identName(name string) string {
	if name == "" {
		return ""
	}

	ok := true
	for i, r := range name {
		if !unicode.IsLetter(r) && (i == 0 || !unicode.IsDigit(r)) {
			ok = false
			break
		}
	}
	if ok {
		return name
	}

	var sb strings.Builder
	for i, r := range name {
		if !unicode.IsLetter(r) && (i == 0 || !unicode.IsDigit(r)) {
			sb.WriteRune('_')
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func valueTypeKey(t wasm.ValueType) rune {
	switch t {
	case wasm.ValueTypeI32:
		return 'i'
	case wasm.ValueTypeI64:
		return 'I'
	case wasm.ValueTypeF32:
		return 'f'
	case wasm.ValueTypeF64:
		return 'F'
	default:
		panic("unreachable")
	}
}

func functionTypeKey(params, results []wasm.ValueType) string {
	var b strings.Builder
	b.WriteRune('p')
	for _, t := range params {
		b.WriteRune(valueTypeKey(t))
	}
	b.WriteRune('r')
	for _, t := range results {
		b.WriteRune(valueTypeKey(t))
	}
	return b.String()
}

type formatter struct {
	w   io.Writer
	buf bytes.Buffer
}

func (f *formatter) flush() error {
	// Format code
	bytes, err := format.Source(f.buf.Bytes())
	if err != nil {
		f.w.Write(f.buf.Bytes())
		return err
	}
	_, err = f.w.Write(bytes)
	return err
}

func (f *formatter) Write(b []byte) (int, error) {
	return f.buf.Write(b)
}

// Format returns a Writer that formats Go source code prior to emitting it.
func Format(w io.Writer) io.Writer {
	return &formatter{w: w}
}

// Options records compilation options.
type Options struct {
	// UseRawPointers enables the use of raw pointers in place of calls to exec.Memory methods for
	// loads and stores.
	UseRawPointers bool
	// NoInternalThreads disables the use of *exec.Thread inside the generated code.
	NoInternalThreads bool
}

func (o *Options) apply(m *moduleCompiler) {
	if o != nil {
		m.useRawPointers = o.UseRawPointers
		m.noInternalThreads = o.NoInternalThreads
	}
}

// CompileModule compiles the given module into Go source code and writes the source to
// the given writer. The source will be contained in the named package, and will contain
// an exec.ModuleDefintion with the exported version of the given name.
func CompileModule(w io.Writer, packageName, name string, module *wasm.Module, options *Options) error {
	if err := validate.ValidateModule(module, true); err != nil {
		return err
	}

	name = identName(name)

	compiler := moduleCompiler{
		packageName:  packageName,
		name:         unexportName(name),
		exportedName: exportName(name),
		module:       module,
	}
	options.apply(&compiler)

	compiler.compile()
	return compiler.emit(w)
}

// CompileCommand compiles the given WASI module into Go source code and writes the source
// to the given writer. The source will be contained in package main, and will contain a
// main function.
func CompileCommand(w io.Writer, name string, module *wasm.Module, options *Options) error {
	if err := validate.ValidateModule(module, true); err != nil {
		return err
	}

	compiler := moduleCompiler{
		isCommand:    true,
		packageName:  "main",
		name:         unexportName(name),
		exportedName: exportName(name),
		module:       module,
	}
	options.apply(&compiler)

	compiler.compile()
	return compiler.emit(w)
}

func (m *moduleCompiler) GetLocalType(localidx uint32) (wasm.ValueType, bool) {
	return 0, false
}

func (m *moduleCompiler) GetGlobalType(globalidx uint32) (wasm.GlobalVar, bool) {
	if globalidx < uint32(len(m.importedGlobals)) {
		return m.importedGlobals[int(globalidx)], true
	}
	globalidx -= uint32(len(m.importedGlobals))
	if m.module.Global == nil || globalidx >= uint32(len(m.module.Global.Globals)) {
		return wasm.GlobalVar{}, false
	}
	return m.module.Global.Globals[int(globalidx)].Type, true
}

func (m *moduleCompiler) GetFunctionSignature(funcidx uint32) (wasm.FunctionSig, bool) {
	if funcidx < uint32(len(m.importedFunctions)) {
		return m.importedFunctions[int(funcidx)], true
	}
	funcidx -= uint32(len(m.importedFunctions))
	if m.module.Function == nil || funcidx >= uint32(len(m.module.Function.Types)) {
		return wasm.FunctionSig{}, false
	}
	return m.GetType(m.module.Function.Types[int(funcidx)])
}

func (m *moduleCompiler) GetType(typeidx uint32) (wasm.FunctionSig, bool) {
	if m.module.Types == nil || typeidx >= uint32(len(m.module.Types.Entries)) {
		return wasm.FunctionSig{}, false
	}
	return m.module.Types.Entries[int(typeidx)], true
}

func (m *moduleCompiler) HasTable(tableidx uint32) bool {
	return tableidx == 0 && (m.importedTable != nil || m.module.Table != nil && len(m.module.Table.Entries) != 0)
}

func (m *moduleCompiler) HasMemory(memoryidx uint32) bool {
	return memoryidx == 0 && (m.importedMemory != nil || m.module.Memory != nil && len(m.module.Memory.Entries) != 0)
}

func (m *moduleCompiler) globalType(globalidx uint32) wasm.ValueType {
	if globalidx < uint32(len(m.importedGlobals)) {
		return m.importedGlobals[int(globalidx)].Type
	}
	return m.module.Global.Globals[int(globalidx)-len(m.importedGlobals)].Type.Type
}

func (m *moduleCompiler) compile() {
	// Record import counts for index spaces
	if m.module.Import != nil {
		for i, import_ := range m.module.Import.Entries {
			switch type_ := import_.Type.(type) {
			case wasm.FuncImport:
				m.importedFunctions = append(m.importedFunctions, m.module.Types.Entries[int(type_.Type)])
			case wasm.MemoryImport:
				m.importedMemory = &m.module.Import.Entries[i]
			case wasm.TableImport:
				m.importedTable = &m.module.Import.Entries[i]
			case wasm.GlobalVarImport:
				m.importedGlobals = append(m.importedGlobals, type_.Type)
			}
		}
	}

	// Record exports for global accesses + thunks
	if m.module.Export != nil {
		m.exportedFunctions, m.exportedGlobals = map[uint32]bool{}, map[uint32]bool{}
		for _, export := range m.module.Export.Entries {
			switch export.Kind {
			case wasm.ExternalFunction:
				m.exportedFunctions[export.Index] = true
			case wasm.ExternalGlobal:
				m.exportedGlobals[export.Index] = true
			}
		}
	}

	// Record function names if available.
	if names, err := m.module.Names(); err == nil {
		for _, entry := range names.Entries {
			if entry, ok := entry.(*wasm.FunctionNamesSubsection); ok {
				m.functionNames = map[uint32]string{}
				for _, name := range entry.Names {
					m.functionNames[name.Index] = identName(name.Name)
				}
			}
		}
	}

	// Compile functions
	if m.module.Code != nil {
		m.functions = make([]functionCompiler, len(m.module.Code.Bodies))
		for i, body := range m.module.Code.Bodies {
			funcidx := i + len(m.importedFunctions)

			typeidx := m.module.Function.Types[i]
			m.functions[i].compile(m, funcidx, typeidx, m.module.Types.Entries[typeidx], body)
		}
	}
}

func (m *moduleCompiler) functionName(index uint32) string {
	if name, ok := m.functionNames[index]; ok {
		return fmt.Sprintf("f%d_%v", index, name)
	}
	return fmt.Sprintf("%s_f%d", m.name, index)
}

func (m *moduleCompiler) functionTypeName(sig wasm.FunctionSig) string {
	return fmt.Sprintf("%s_f%s", m.name, functionTypeKey(sig.ParamTypes, sig.ReturnTypes))
}

func (m *moduleCompiler) typeName(typeidx uint32) string {
	return m.functionTypeName(m.module.Types.Entries[int(typeidx)])
}

func (m *moduleCompiler) emit(w io.Writer) error {
	// Commands must have a _start function with the signature [] -> []
	if m.isCommand {
		hasEntrypoint := false
		if m.module.Export != nil {
			for _, export := range m.module.Export.Entries {
				if export.Kind == wasm.ExternalFunction && export.FieldStr == "_start" {
					sig := m.module.Types.Entries[m.module.Function.Types[int(export.Index)-len(m.importedFunctions)]]
					if !sig.Equals(wasm.FunctionSig{}) {
						return fmt.Errorf("_start must not accept or return parameters")
					}
					hasEntrypoint = true
					break
				}
			}
		}
		if !hasEntrypoint {
			return fmt.Errorf("missing _start function")
		}
	}

	// Emit package declaration and imports
	if err := m.emitPackage(w); err != nil {
		return err
	}

	// Emit function types
	if err := m.emitFunctionTypes(w); err != nil {
		return err
	}

	// Emit module definition
	if err := m.emitModuleDefinition(w); err != nil {
		return err
	}

	// Emit module
	if err := m.emitModule(w); err != nil {
		return err
	}

	// Emit main
	if m.isCommand {
		if err := m.emitMain(w); err != nil {
			return err
		}
	}

	if f, ok := w.(*formatter); ok {
		return f.flush()
	}
	return nil
}

func (m *moduleCompiler) emitPackage(w io.Writer) error {
	t := template.Must(template.New("Package").Parse(`package {{.PackageName}}

import (
	{{range .Imports -}}
	"{{.}}"
	{{end -}}
)
`))

	imports := []string{
		"math",
		"math/bits",
		"unsafe",
		"github.com/pgavlin/warp/exec",
		"github.com/pgavlin/warp/wasm",
	}
	if m.isCommand {
		imports = append(imports, "github.com/pgavlin/warp/wasi")
	}

	return t.Execute(w, map[string]interface{}{
		"PackageName": m.packageName,
		"Imports":     imports,
	})
}

func (m *moduleCompiler) emitFunctionTypes(w io.Writer) error {
	if m.module.Types == nil {
		return nil
	}

	emitted := map[string]bool{}
	for i, t := range m.module.Types.Entries {
		name := m.functionTypeName(t)
		if emitted[name] {
			continue
		}
		emitted[name] = true

		if err := m.emitFunctionType(w, t, uint32(i), name); err != nil {
			return err
		}
	}
	return nil
}

func (m *moduleCompiler) emitModuleDefinition(w io.Writer) error {
	t := template.Must(template.New("ModuleDefinition").Parse(`var {{.ExportedName}} = &{{.Name}}{
	types: {{printf "%#v" .Types}},
}

type {{.Name}} struct {
	types []wasm.FunctionSig
}

func ({{.Name}}) Allocate(name string) (exec.AllocatedModule, error) {
	return allocate{{.ExportedName}}(name)
}
`))

	var imports []wasm.ImportEntry
	if m.module.Import != nil {
		imports = m.module.Import.Entries
	}
	var types []wasm.FunctionSig
	if m.module.Types != nil {
		types = m.module.Types.Entries
	}
	return t.Execute(w, map[string]interface{}{
		"Name":         m.name,
		"ExportedName": m.exportedName,
		"Imports":      imports,
		"Types":        types,
	})
}

func (m *moduleCompiler) emitModule(w io.Writer) error {
	if err := m.emitModuleType(w); err != nil {
		return err
	}
	if err := m.emitAllocatedModule(w); err != nil {
		return err
	}
	if err := m.emitInitFunctions(w); err != nil {
		return err
	}
	if err := m.emitInitGlobals(w); err != nil {
		return err
	}
	if err := m.emitInitSections(w); err != nil {
		return err
	}
	if err := m.emitGetters(w); err != nil {
		return err
	}
	if err := m.emitHelpers(w); err != nil {
		return err
	}
	return m.emitFunctions(w)
}

func (m *moduleCompiler) emitModuleType(w io.Writer) error {
	t := template.Must(template.New("Module").Parse(`type {{.Name}}Instance struct {
	name string

	mem0   *exec.Memory
	table0 *exec.Table

	mem    uintptr
	table  []exec.Function

	importedFunctions []exec.Function
	importedGlobals   []*exec.Global

	exports map[string]interface{}

	{{range .Globals -}}
	g{{.Index}} {{.Type}}
	{{end -}}
}
`))

	type global struct {
		Index uint32
		Type  string
	}
	var globals []global
	for i := range m.importedGlobals {
		globals = append(globals, global{Index: uint32(i), Type: "*exec.Global"})
	}
	if m.module.Global != nil {
		for i, g := range m.module.Global.Globals {
			gg := global{Index: uint32(len(m.importedGlobals) + i), Type: goType(g.Type.Type)}
			if m.exportedGlobals[gg.Index] {
				gg.Type = "exec.Global"
			}
			globals = append(globals, gg)
		}
	}
	return t.Execute(w, map[string]interface{}{
		"Name":    m.name,
		"Globals": globals,
	})
}

func (m *moduleCompiler) emitAllocatedModule(w io.Writer) error {
	t := template.Must(template.New("AllocatedModule").Parse(`type allocated{{.ExportedName}} struct {
	*{{.Name}}Instance
}

{{- $moduleName := .Name }}

func allocate{{.ExportedName}}(name string) (exec.AllocatedModule, error) {
	m := &{{.Name}}Instance{
		name: name,
	}

	{{if .NewMem0 -}}
	mem0 := exec.NewMemory({{.MinMem0}}, {{.MaxMem0}})
	m.mem0 = &mem0
	{{- end}}

	{{if .NewTable0 -}}
	table0 := exec.NewTable({{.MinTable0}}, {{.MaxTable0}})
	m.table0 = &table0
	{{- end}}

	{{range .Globals -}}
	{{if .Exported -}}
	m.g{{.Index}} = exec.NewGlobal{{.Type}}({{.Immutable}}, 0)
	{{- end}}
	{{end -}}

	{{if .HasExports -}}
	m.exports = map[string]interface{}{}
	{{if .ExportMem0 -}}
	m.exports[{{printf "%q" .ExportMem0.FieldStr}}] = m.mem0
	{{- end}}
	{{if .ExportTable0 -}}
	m.exports[{{printf "%q" .ExportTable0.FieldStr}}] = m.table0
	{{- end}}
	{{range .ExportedGlobals -}}
	{{if not .Imported -}}
	m.exports[{{printf "%q" .FieldStr}}] = &m.g{{.Index}}
	{{- end}}
	{{end -}}
	{{range .ExportedFunctions -}}
	{{if not .Imported -}}
	m.exports[{{printf "%q" .FieldStr}}] = new{{.TypeName}}(m, {{.Name}})
	{{- end}}
	{{end -}}
	{{- end}}

	return &allocated{{.ExportedName}}{
		{{.Name}}Instance: m,
	}, nil
}

func (m *allocated{{.ExportedName}}) Instantiate(imports exec.ImportResolver) (exec.Module, error) {
	if err := m.initFunctions(imports); err != nil {
		return nil, err
	}

	{{with .ImportMem0 -}}
	mem, err := imports.ResolveMemory({{printf "%q" .ModuleName}}, {{printf "%q" .FieldName}}, {{printf "%#v" .Type}})
	if err != nil {
		return nil, err
	}
	m.mem0 = mem
	{{- end}}

	{{with .ImportTable0 -}}
	table, err := imports.ResolveTable({{printf "%q" .ModuleName}}, {{printf "%q" .FieldName}}, {{printf "%#v" .Type}})
	if err != nil {
		return nil, err
	}
	m.table0 = table
	{{- end}}

	if err := m.initGlobals(imports); err != nil {
		return nil, err
	}

	if err := m.checkOffsets(); err != nil {
		return nil, err
	}
	{{if or .ImportTable0 .NewTable0 -}}
	m.table = m.table0.Entries()
	{{- end}}
	m.initTable()
	{{if and .UseRawPointers (or .ImportMem0 .NewMem0) -}}
	m.mem = m.mem0.Start()
	{{- end}}
	m.initMemory()

	{{if .HasExports -}}
	{{if .ExportMem0 -}}
	m.exports[{{printf "%q" .ExportMem0.FieldStr}}] = m.mem0
	{{- end}}
	{{if .ExportTable0 -}}
	m.exports[{{printf "%q" .ExportTable0.FieldStr}}] = m.table0
	{{- end}}
	{{range .ExportedGlobals -}}
	{{if .Imported}}
	m.exports[{{printf "%q" .FieldStr}}] = m.g{{.Index}}
	{{- end}}
	{{end -}}
	{{range .ExportedFunctions -}}
	{{if .Imported -}}
	m.exports[{{printf "%q" .FieldStr}}] = m.importedFunctions[{{.Index}}]
	{{- end}}
	{{end -}}
	{{- end}}

	{{if .HasStart -}}
	t := exec.NewThread(0)
	{{.StartName}}(m.{{$moduleName}}Instance, &t)
	{{- end}}

	return m, nil
}

`))

	type memImport struct {
		ModuleName string
		FieldName  string
		Type       wasm.Memory
	}

	importMem0, newMem0, minMem0, maxMem0 := (*memImport)(nil), false, uint32(0), uint32(0)
	if m.importedMemory != nil {
		importMem0 = &memImport{
			ModuleName: m.importedMemory.ModuleName,
			FieldName:  m.importedMemory.FieldName,
			Type:       m.importedMemory.Type.(wasm.MemoryImport).Type,
		}
	} else if m.module.Memory != nil && len(m.module.Memory.Entries) != 0 {
		newMem0 = true
		mem0Def := m.module.Memory.Entries[0]
		minMem0 = mem0Def.Limits.Initial
		maxMem0 = mem0Def.Limits.Maximum
		if mem0Def.Limits.Flags == 0 {
			maxMem0 = 65536
		}
	}

	type tableImport struct {
		ModuleName string
		FieldName  string
		Type       wasm.Table
	}

	importTable0, newTable0, minTable0, maxTable0 := (*tableImport)(nil), false, uint32(0), uint32(0)
	if m.importedTable != nil {
		importTable0 = &tableImport{
			ModuleName: m.importedTable.ModuleName,
			FieldName:  m.importedTable.FieldName,
			Type:       m.importedTable.Type.(wasm.TableImport).Type,
		}
	} else if m.module.Table != nil && len(m.module.Table.Entries) != 0 {
		newTable0 = true
		table0Def := m.module.Table.Entries[0]
		minTable0 = table0Def.Limits.Initial
		maxTable0 = table0Def.Limits.Maximum
		if table0Def.Limits.Flags == 0 {
			maxTable0 = ^uint32(0)
		}
	}

	type functionExport struct {
		wasm.ExportEntry

		Name     string
		Imported bool
		TypeName string
	}

	type globalExport struct {
		wasm.ExportEntry

		Imported bool
	}

	hasExports, exportMem0, exportTable0, exportedGlobals, exportedFunctions := m.module.Export != nil, (*wasm.ExportEntry)(nil), (*wasm.ExportEntry)(nil), []globalExport(nil), []functionExport(nil)
	if m.module.Export != nil {
		for _, export := range m.module.Export.Entries {
			switch export.Kind {
			case wasm.ExternalFunction:
				fx := functionExport{ExportEntry: export, Name: m.functionName(export.Index)}
				if export.Index < uint32(len(m.importedFunctions)) {
					fx.Imported = true
				} else {
					fx.TypeName = exportName(m.typeName(m.module.Function.Types[int(export.Index)-len(m.importedFunctions)]))
				}
				exportedFunctions = append(exportedFunctions, fx)
			case wasm.ExternalMemory:
				if export.Index != 0 {
					return ErrInvalidMemoryIndex
				}
				x := export
				exportMem0 = &x
			case wasm.ExternalTable:
				if export.Index != 0 {
					return exec.InvalidTableIndexError(export.Index)
				}
				x := export
				exportTable0 = &x
			case wasm.ExternalGlobal:
				exportedGlobals = append(exportedGlobals, globalExport{
					ExportEntry: export,
					Imported:    export.Index < uint32(len(m.importedGlobals)),
				})
			}
		}
	}

	hasStart, startName := false, ""
	if m.module.Start != nil {
		hasStart, startName = true, m.functionName(m.module.Start.Index)
	}

	return t.Execute(w, map[string]interface{}{
		"Name":              m.name,
		"ExportedName":      m.exportedName,
		"UseRawPointers":    m.useRawPointers,
		"ImportMem0":        importMem0,
		"NewMem0":           newMem0,
		"MinMem0":           minMem0,
		"MaxMem0":           maxMem0,
		"ImportTable0":      importTable0,
		"NewTable0":         newTable0,
		"MinTable0":         minTable0,
		"MaxTable0":         maxTable0,
		"HasExports":        hasExports,
		"ExportMem0":        exportMem0,
		"ExportTable0":      exportTable0,
		"ExportedGlobals":   exportedGlobals,
		"ExportedFunctions": exportedFunctions,
		"HasStart":          hasStart,
		"StartName":         startName,
	})
}

func (m *moduleCompiler) emitInitFunctions(w io.Writer) error {
	t := template.Must(template.New("InitFunctions").Parse(`func (m *{{.Name}}Instance) initFunctions(imports exec.ImportResolver) (err error) {
	{{if .Functions -}}
	m.importedFunctions = make([]exec.Function, {{len .Functions}})
	{{end -}}
	{{range $i, $f := .Functions -}}
	m.importedFunctions[{{$i}}], err = imports.ResolveFunction({{printf "%q" $f.ModuleName}}, {{printf "%q" $f.FieldName}}, {{printf "%#v" $f.Signature}})
	if err != nil {
		return err
	}
	{{end -}}
	return nil
}

`))

	type function struct {
		ModuleName string
		FieldName  string
		Signature  wasm.FunctionSig
	}
	functions := make([]function, 0, len(m.importedFunctions))
	if m.module.Import != nil {
		for _, import_ := range m.module.Import.Entries {
			if _, ok := import_.Type.(wasm.FuncImport); ok {
				functions = append(functions, function{
					ModuleName: import_.ModuleName,
					FieldName:  import_.FieldName,
					Signature:  m.importedFunctions[len(functions)],
				})
			}
		}
	}
	return t.Execute(w, map[string]interface{}{
		"Name":      m.name,
		"Functions": functions,
	})
}

func (m *moduleCompiler) emitInitGlobals(w io.Writer) error {
	t := template.Must(template.New("InitGlobals").Parse(`func (m *{{.Name}}Instance) initGlobals(imports exec.ImportResolver) (err error) {
	{{range .Globals -}}
	{{if .Imported -}}
	m.g{{.Index}}, err = imports.ResolveGlobal({{printf "%q" .ModuleName}}, {{printf "%q" .FieldName}}, {{printf "%#v" .ImportType}})
	if err != nil {
		return err
	}
	{{- else if .Exported -}}
	m.g{{.Index}} = exec.NewGlobal{{.Type}}({{.Immutable}}, {{.Value}})
	{{- else -}}
	m.g{{.Index}} = {{.Value}}
	{{- end}}
	{{end -}}
	return nil
}

`))

	type global struct {
		Imported   bool
		ModuleName string
		FieldName  string
		ImportType wasm.GlobalVar
		Exported   bool
		Index      uint32
		Type       string
		Immutable  bool
		Value      interface{}
	}

	var globals []global
	if m.module.Import != nil {
		for _, import_ := range m.module.Import.Entries {
			if _, ok := import_.Type.(wasm.GlobalVarImport); ok {
				i := uint32(len(globals))
				globals = append(globals, global{
					Imported:   true,
					ModuleName: import_.ModuleName,
					FieldName:  import_.FieldName,
					ImportType: import_.Type.(wasm.GlobalVarImport).Type,
					Exported:   m.exportedGlobals[i],
					Index:      i,
				})
			}
		}
	}
	if m.module.Global != nil {
		for i, g := range m.module.Global.Globals {
			body, err := code.Decode(g.Init, m, []wasm.ValueType{g.Type.Type})
			if err != nil {
				return err
			}
			c := constExpressionCompiler{m: m, code: body.Instructions}
			c.compile()
			_, value := c.emit()

			gg := global{Index: uint32(len(m.importedGlobals) + i), Immutable: !g.Type.Mutable, Value: value}
			if m.exportedGlobals[gg.Index] {
				gg.Exported = true
				switch g.Type.Type {
				case wasm.ValueTypeI32:
					gg.Type = "I32"
				case wasm.ValueTypeI64:
					gg.Type = "I64"
				case wasm.ValueTypeF32:
					gg.Type = "F32"
				case wasm.ValueTypeF64:
					gg.Type = "F64"
				default:
					panic("unreachable")
				}
			}
			globals = append(globals, gg)
		}
	}
	return t.Execute(w, map[string]interface{}{
		"Name":    m.name,
		"Globals": globals,
	})
}

func (m *moduleCompiler) emitInitSections(w io.Writer) error {
	elementOffsets, dataOffsets, err := m.emitCheckOffsets(w)
	switch {
	case err == exec.ErrElementSegmentDoesNotFit:
		return m.emitInitSectionsError(w, "exec.ErrElementSegmentDoesNotFit")
	case err == exec.ErrDataSegmentDoesNotFit:
		return m.emitInitSectionsError(w, "exec.ErrDataSegmentDoesNotFit")
	case err != nil:
		return err
	}
	if err := m.emitInitTable(w, elementOffsets); err != nil {
		return err
	}
	return m.emitInitMemory(w, dataOffsets)
}

func (m *moduleCompiler) emitInitSectionsError(w io.Writer, err string) error {
	t := template.Must(template.New("InitSectionsError").Parse(`func (m *{{.Name}}Instance) checkOffsets() error {
	return {{.Error}}
}

func (m *{{.Name}}Instance) initTable() {
}

func (m *{{.Name}}Instance) initMemory() {
}

`))

	return t.Execute(w, map[string]interface{}{
		"Name":  m.name,
		"Error": err,
	})
}

func (m *moduleCompiler) emitCheckOffsets(w io.Writer) ([]string, []string, error) {
	t := template.Must(template.New("CheckOffsets").Parse(`func (m *{{.Name}}Instance) checkOffsets() error {
	{{if .Elements -}}
	table := m.table0.Entries()
	{{range $i, $e := .Elements -}}
	if int32(len(table)) < {{$e.Offset}} || len(table[int({{$e.Offset}}):]) < {{len $e.Elems}} {
		return exec.ErrElementSegmentDoesNotFit
	}
	{{end -}}
	{{- end}}

	{{if .Data -}}
	bytes := m.mem0.Bytes()
	{{range $i, $e := .Data -}}
	if int32(len(bytes)) < {{$e.Offset}} || len(bytes[int({{$e.Offset}}):]) < {{len $e.Data}} {
		return exec.ErrDataSegmentDoesNotFit
	}
	{{end -}}
	{{- end}}

	return nil
}

`))

	var elementOffsets []string
	var dataOffsets []string

	type element struct {
		Offset string
		Elems  []uint32
	}
	var elements []element
	if m.module.Elements != nil {
		for _, e := range m.module.Elements.Entries {
			body, err := code.Decode(e.Offset, m, []wasm.ValueType{wasm.ValueTypeI32})
			if err != nil {
				return nil, nil, err
			}
			c := constExpressionCompiler{m: m, code: body.Instructions}
			c.compile()

			offset, offsetText := c.emit()
			if offset != nil && offset.(int32) < 0 {
				return nil, nil, exec.ErrElementSegmentDoesNotFit
			}

			elementOffsets = append(elementOffsets, offsetText)
			elements = append(elements, element{
				Offset: offsetText,
				Elems:  e.Elems,
			})
		}
	}

	type data struct {
		Offset string
		Data   []byte
	}
	var datas []data
	if m.module.Data != nil {
		for _, e := range m.module.Data.Entries {
			body, err := code.Decode(e.Offset, m, []wasm.ValueType{wasm.ValueTypeI32})
			if err != nil {
				return nil, nil, err
			}
			c := constExpressionCompiler{m: m, code: body.Instructions}
			c.compile()

			offset, offsetText := c.emit()
			if offset != nil && offset.(int32) < 0 {
				return nil, nil, exec.ErrDataSegmentDoesNotFit
			}

			dataOffsets = append(dataOffsets, offsetText)
			datas = append(datas, data{
				Offset: offsetText,
				Data:   e.Data,
			})
		}
	}

	return elementOffsets, dataOffsets, t.Execute(w, map[string]interface{}{
		"Name":     m.name,
		"Elements": elements,
		"Data":     datas,
	})
}

func (m *moduleCompiler) emitInitTable(w io.Writer, offsets []string) error {
	t := template.Must(template.New("InitTable").Parse(`func (m *{{.Name}}Instance) initTable() {
	{{$moduleName := .Name}}
	{{if .Elements -}}
	{{range $i, $e := .Elements -}}
	{{range $j, $chunk := $e.Chunks -}}
	{{$moduleName}}_initTable_{{$i}}_{{$j}}(m)
	{{end -}}
	{{end -}}
	{{- end}}
}

{{if .Elements -}}
{{range $i, $e := .Elements -}}
{{range $j, $chunk := $e.Chunks -}}
func {{$moduleName}}_initTable_{{$i}}_{{$j}}(m *{{$moduleName}}Instance) {
	start := int({{$chunk.Offset}}) + {{$chunk.Start}}
	end := start + {{len $chunk.Elems}}
	table := m.table0.Entries()[start:end]
	{{range $k, $expr := $chunk.Elems -}}
	table[{{$k}}] = {{$expr}}
	{{end -}}
}
{{end -}}
{{end -}}
{{- end}}
`))

	type chunk struct {
		Offset string
		Start  int
		Elems  []string
	}

	type element struct {
		Chunks []chunk
	}
	var elements []element
	if m.module.Elements != nil {
		for i, e := range m.module.Elements.Entries {
			if len(e.Elems) == 0 {
				continue
			}

			offset := offsets[i]
			nChunks := len(e.Elems) / 256
			if len(e.Elems)%256 != 0 {
				nChunks++
			}
			chunks := make([]chunk, nChunks)
			for i := range chunks {
				start := i * 256
				end := start + 256
				if end > len(e.Elems) {
					end = len(e.Elems)
				}
				indices := e.Elems[start:end]
				elems := make([]string, len(indices))
				for j, funcidx := range indices {
					typeName := ""
					if funcidx < uint32(len(m.importedFunctions)) {
						typeName = m.functionTypeName(m.importedFunctions[funcidx])
					} else {
						typeName = m.typeName(m.module.Function.Types[funcidx-uint32(len(m.importedFunctions))])
					}

					elems[j] = fmt.Sprintf("new%s(m, %s)", exportName(typeName), m.functionName(funcidx))
				}
				chunks[i] = chunk{Offset: offset, Start: start, Elems: elems}
			}

			elements = append(elements, element{Chunks: chunks})
		}
	}

	return t.Execute(w, map[string]interface{}{
		"Name":     m.name,
		"Elements": elements,
	})
}

func (m *moduleCompiler) emitInitMemory(w io.Writer, offsets []string) error {
	t := template.Must(template.New("InitMemory").Parse(`func (m *{{.Name}}Instance) initMemory() {
	{{if .Data -}}
	bytes := m.mem0.Bytes()
	{{range $i, $e := .Data -}}
	copy(bytes[{{$e.Offset}}:], {{printf "%#v" $e.Data}})
	{{end -}}
	{{- end}}
}

`))

	type data struct {
		Offset string
		Data   []byte
	}
	var datas []data
	if m.module.Data != nil {
		for i, e := range m.module.Data.Entries {
			if len(e.Data) == 0 {
				continue
			}
			datas = append(datas, data{
				Offset: offsets[i],
				Data:   e.Data,
			})
		}
	}

	return t.Execute(w, map[string]interface{}{
		"Name":              m.name,
		"ImportedFunctions": m.importedFunctions,
		"Data":              datas,
	})
}

func (m *moduleCompiler) emitGetters(w io.Writer) error {
	t := template.Must(template.New("Getters").Parse(`func (m *{{.Name}}Instance) Close() error {
	return nil
}

func (m *{{.Name}}Instance) Name() string {
	return m.name
}

func (m *{{.Name}}Instance) newExportError(name string, importKind wasm.External, export interface{}) error {
	if export == nil {
		return &exec.ExportNotFoundError{ModuleName: m.name, FieldName: name}
	}

	var exportKind wasm.External
	switch export.(type) {
	case exec.Function:
		exportKind = wasm.ExternalFunction
	case *exec.Table:
		exportKind = wasm.ExternalTable
	case *exec.Memory:
		exportKind = wasm.ExternalMemory
	case *exec.Global:
		exportKind = wasm.ExternalGlobal
	default:
		panic("unreachable")
	}
	return exec.NewKindMismatchError(m.name, name, importKind, exportKind)
}

func (m *{{.Name}}Instance) GetFunction(name string) (exec.Function, error) {
	export := m.exports[name]
	if function, ok := export.(exec.Function); ok {
		return function, nil
	}
	return nil, m.newExportError(name, wasm.ExternalFunction, export)
}

func (m *{{.Name}}Instance) GetTable(name string) (*exec.Table, error) {
	export := m.exports[name]
	if table, ok := export.(*exec.Table); ok {
		return table, nil
	}
	return nil, m.newExportError(name, wasm.ExternalFunction, export)
}

func (m *{{.Name}}Instance) GetMemory(name string) (*exec.Memory, error) {
	export := m.exports[name]
	if memory, ok := export.(*exec.Memory); ok {
		return memory, nil
	}
	return nil, m.newExportError(name, wasm.ExternalFunction, export)
}

func (m *{{.Name}}Instance) GetGlobal(name string) (*exec.Global, error) {
	export := m.exports[name]
	if global, ok := export.(*exec.Global); ok {
		return global, nil
	}
	return nil, m.newExportError(name, wasm.ExternalFunction, export)
}

`))
	return t.Execute(w, map[string]interface{}{"Name": m.name})
}

func (m *moduleCompiler) emitHelpers(w io.Writer) error {
	t := template.Must(template.New("Helpers").Parse(`func (m *{{.Name}}Instance) tableEntry(tableidx uint32) exec.Function {
	if tableidx >= uint32(len(m.table)) {
		panic(exec.TrapUndefinedElement)
	}
	return m.table[tableidx]
}

func (m *{{.Name}}Instance) callFunction({{.ThreadParam}}function exec.Function, typeidx uint32, args, results []uint64) {
	expectedSig := {{.ExportedName}}.types[int(typeidx)]
	actualSig := function.GetSignature()
	if !actualSig.Equals(expectedSig) {
		panic(exec.TrapIndirectCallTypeMismatch)
	}

	{{if not .ThreadParam -}}
	thread := exec.NewThread(0)
	t := &thread
	{{- end}}
	function.UncheckedCall(t, args, results)
}

func (m *{{.Name}}Instance) i32Bool(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

var _ = math.MaxInt32
var _ = bits.UintSize
var _ = unsafe.Pointer(uintptr(0))
`))

	threadParam := "t *exec.Thread, "
	if m.noInternalThreads {
		threadParam = ""
	}

	return t.Execute(w, map[string]interface{}{
		"Name":         m.name,
		"ExportedName": m.exportedName,
		"ThreadParam":  threadParam,
	})
}

func (m *moduleCompiler) emitFunctions(w io.Writer) error {
	for i, f := range m.importedFunctions {
		if err := m.emitImportedFunction(w, uint32(i), f); err != nil {
			return err
		}
	}

	for _, f := range m.functions {
		if err := f.emit(w); err != nil {
			return err
		}
	}
	return nil
}

func (m *moduleCompiler) emitMain(w io.Writer) error {
	t := template.Must(template.New("Main").Parse(`func main() {
	wasi.Main({{.ExportedName}})
}
`))

	return t.Execute(w, map[string]interface{}{"ExportedName": m.exportedName})
}

func (m *moduleCompiler) emitFunctionType(w io.Writer, sig wasm.FunctionSig, typeidx uint32, name string) error {
	// Emit the function type.
	if err := printf(w, "type %s struct {\n\tm *%sInstance\n\tf func", name, m.name); err != nil {
		return err
	}
	if err := m.emitFunctionSignature(w, sig, false); err != nil {
		return err
	}
	if err := printf(w, "\n}\n\n"); err != nil {
		return err
	}

	// Emit the factory function.
	if err := m.emitFactoryFunction(w, sig, name); err != nil {
		return err
	}

	// Emit the indirect call function.
	if err := m.emitIndirectCallFunction(w, sig, typeidx, name); err != nil {
		return err
	}

	// Emit the `GetSignature` function.
	if err := printf(w, "func (f *%s) GetSignature() wasm.FunctionSig {\n\treturn %#v\n}\n\n", name, sig); err != nil {
		return err
	}

	// Emit the `Call` function.
	if err := emitCallFunction(w, sig, name, m.noInternalThreads); err != nil {
		return err
	}

	// Emit the `UncheckedCall` function.
	return emitUncheckedCallFunction(w, sig, name, m.noInternalThreads)
}

func (m *moduleCompiler) emitIndirectCallFunction(w io.Writer, sig wasm.FunctionSig, typeidx uint32, name string) error {
	if err := printf(w, "func %sCallIndirect", name); err != nil {
		return err
	}
	if err := m.emitFunctionSignature(w, sig, true); err != nil {
		return err
	}
	if err := printf(w, " {\n\tfunction := m.tableEntry(tableidx)\n\tif f, ok := function.(*%s); ok {\n\t\t", name); err != nil {
		return err
	}

	threadArg := ", t"
	if m.noInternalThreads {
		threadArg = ""
	}

	if len(sig.ReturnTypes) > 0 {
		if err := printf(w, "return "); err != nil {
			return err
		}
	}
	if err := printf(w, "f.f(f.m%s", threadArg); err != nil {
		return err
	}
	for i := range sig.ParamTypes {
		if err := printf(w, ", v%d", i); err != nil {
			return err
		}
	}
	if err := printf(w, ")\n"); err != nil {
		return err
	}
	if len(sig.ReturnTypes) == 0 {
		if err := printf(w, "\t\treturn\n"); err != nil {
			return err
		}
	}
	if err := printf(w, "\t}\n"); err != nil {
		return err
	}

	args := "nil"
	if len(sig.ParamTypes) > 0 {
		if err := printf(w, "\tca := [...]uint64{"); err != nil {
			return err
		}
		for i, t := range sig.ParamTypes {
			var err error
			switch t {
			case wasm.ValueTypeI32, wasm.ValueTypeI64:
				err = printf(w, "%vuint64(v%d)", comma(i), i)
			case wasm.ValueTypeF32:
				err = printf(w, "%vuint64(math.Float32bits(float32(v%d)))", comma(i), i)
			case wasm.ValueTypeF64:
				err = printf(w, "%vmath.Float64bits(v%d)", comma(i), i)
			default:
				panic("unknown value type")
			}
			if err != nil {
				return err
			}
		}
		if err := printf(w, "}\n"); err != nil {
			return err
		}
		args = "ca[:]"
	}

	threadArg = "t, "
	if m.noInternalThreads {
		threadArg = ""
	}

	returns := "nil"
	if len(sig.ReturnTypes) > 0 {
		if err := printf(w, "var cr [%d]uint64\n", len(sig.ReturnTypes)); err != nil {
			return err
		}
		returns = "cr[:]"
	}

	if err := printf(w, "m.callFunction(%sfunction, %d, %s, %s)\n", threadArg, typeidx, args, returns); err != nil {
		return err
	}

	if len(sig.ReturnTypes) > 0 {
		if err := printf(w, "\treturn "); err != nil {
			return err
		}
		for i, t := range sig.ReturnTypes {
			var err error
			switch t {
			case wasm.ValueTypeI32:
				err = printf(w, "%vint32(cr[%d])", comma(i), i)
			case wasm.ValueTypeI64:
				err = printf(w, "%vint64(cr[%d])", comma(i), i)
			case wasm.ValueTypeF32:
				err = printf(w, "%vmath.Float32frombits(uint32(cr[%d]))", comma(i), i)
			case wasm.ValueTypeF64:
				err = printf(w, "%vmath.Float64frombits(cr[%d])", comma(i), i)
			default:
				panic("unknown value type")
			}
			if err != nil {
				return err
			}
		}
	}

	return printf(w, "}\n\n")
}

func (m *moduleCompiler) emitFactoryFunction(w io.Writer, sig wasm.FunctionSig, name string) error {
	if err := printf(w, "func new%s(m *%sInstance, f func", exportName(name), m.name); err != nil {
		return err
	}
	if err := m.emitFunctionSignature(w, sig, false); err != nil {
		return err
	}
	if err := printf(w, ") exec.Function {\n"); err != nil {
		return err
	}

	return printf(w, "\treturn &%s{m: m, f: f}\n}\n\n", name)
}

func emitCallFunction(w io.Writer, sig wasm.FunctionSig, name string, noInternalThreads bool) error {
	if err := printf(w, "func (f *%s) Call(t *exec.Thread, a ...interface{}) (r []interface{}) {\n\t", name); err != nil {
		return err
	}
	if err := emitTrapGuard(w); err != nil {
		return err
	}
	if err := printf(w, "\t"); err != nil {
		return err
	}

	threadArg := ", t"
	if noInternalThreads {
		threadArg = ""
	}

	if len(sig.ReturnTypes) > 0 {
		if err := printf(w, "r = make([]interface{}, %d)\n\t", len(sig.ReturnTypes)); err != nil {
			return err
		}
		for i := range sig.ReturnTypes {
			if err := printf(w, "%vr[%d]", comma(i), i); err != nil {
				return err
			}
		}
		if err := printf(w, " = "); err != nil {
			return err
		}
	}
	if err := printf(w, "f.f(f.m%s", threadArg); err != nil {
		return err
	}
	for i, t := range sig.ParamTypes {
		var err error
		switch t {
		case wasm.ValueTypeI32:
			err = printf(w, ", a[%d].(int32)", i)
		case wasm.ValueTypeI64:
			err = printf(w, ", a[%d].(int64)", i)
		case wasm.ValueTypeF32:
			err = printf(w, ", a[%d].(float32)", i)
		case wasm.ValueTypeF64:
			err = printf(w, ", a[%d].(float64)", i)
		default:
			panic("unknown value type")
		}
		if err != nil {
			return err
		}
	}
	return printf(w, ")\n\treturn\n}\n\n")
}

func emitUncheckedCallFunction(w io.Writer, sig wasm.FunctionSig, name string, noInternalThreads bool) error {
	threadArg := ", t"
	if noInternalThreads {
		threadArg = ""
	}

	if err := printf(w, "func (f *%s) UncheckedCall(t *exec.Thread, a, r []uint64) {\n", name); err != nil {
		return err
	}
	if err := emitTrapGuard(w); err != nil {
		return err
	}
	if err := printf(w, "\t"); err != nil {
		return err
	}
	if len(sig.ReturnTypes) > 0 {
		for i := range sig.ReturnTypes {
			if err := printf(w, "%vv%d", comma(i), i); err != nil {
				return err
			}
		}
		if err := printf(w, " := "); err != nil {
			return err
		}
	}
	if err := printf(w, "f.f(f.m%s", threadArg); err != nil {
		return err
	}
	for i, t := range sig.ParamTypes {
		var err error
		switch t {
		case wasm.ValueTypeI32:
			err = printf(w, ", int32(a[%d])", i)
		case wasm.ValueTypeI64:
			err = printf(w, ", int64(a[%d])", i)
		case wasm.ValueTypeF32:
			err = printf(w, ", math.Float32frombits(uint32(a[%d]))", i)
		case wasm.ValueTypeF64:
			err = printf(w, ", math.Float64frombits(a[%d])", i)
		default:
			panic("unknown value type")
		}
		if err != nil {
			return err
		}
	}
	if err := printf(w, ")\n"); err != nil {
		return err
	}
	if len(sig.ReturnTypes) > 0 {
		if err := printf(w, "\t"); err != nil {
			return err
		}
		for i := range sig.ReturnTypes {
			if err := printf(w, "%vr[%d]", comma(i), i); err != nil {
				return err
			}
		}
		if err := printf(w, " = "); err != nil {
			return err
		}
		for i, t := range sig.ReturnTypes {
			var err error
			switch t {
			case wasm.ValueTypeI32, wasm.ValueTypeI64:
				err = printf(w, "%vuint64(v%d)", comma(i), i)
			case wasm.ValueTypeF32:
				err = printf(w, "%vuint64(math.Float32bits(v%d))", comma(i), i)
			case wasm.ValueTypeF64:
				err = printf(w, "%vmath.Float64bits(v%d)", comma(i), i)
			default:
				panic("unknown value type")
			}
			if err != nil {
				return err
			}
		}
	}

	return printf(w, "}\n\n")
}

func emitTrapGuard(w io.Writer) error {
	return printf(w, `	defer func() { exec.TranslateRecover(recover()) }()
`)
}
