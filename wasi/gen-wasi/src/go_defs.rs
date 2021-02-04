use heck::{CamelCase, MixedCase};
use std::collections::HashMap;
use std::mem;
use witx::*;

pub struct Generated {
    pub module: String,
    pub types: String,
    pub stubs: String,
}

pub fn to_go(doc: &Document, inputs_str: &str) -> Generated {
    let mut module = String::new();
    module.push_str(&format!(
        r#"// THIS FILE IS AUTO-GENERATED from the following files:
//
//   {}
//
// To regenerate this file execute:
//
//     cargo run --manifest-path gen-wasi/Cargo.toml generate-api
//
// Modifications to this file will cause CI to fail, the code generator tool
// must be modified to change this file.
//
// This file describes the [WASI] interface, consisting of functions, types,
// and defined values (macros).
//
// The interface described here is greatly inspired by [CloudABI]'s clean,
// thoughtfully-designed, capability-oriented, POSIX-style API.
//
// [CloudABI]: https://github.com/NuxiNL/cloudlibc
// [WASI]: https://github.com/WebAssembly/WASI/

package wasi

import (
    "errors"
    "reflect"

    "github.com/pgavlin/warp/exec"
    "github.com/pgavlin/warp/wasm"
)

"#,
        inputs_str,
    ));

    let mut types = String::new();
    types.push_str(&format!(
        r#"// THIS FILE IS AUTO-GENERATED from the following files:
//
//   {}
//
// To regenerate this file execute:
//
//     cargo run --manifest-path gen-wasi/Cargo.toml generate-api
//
// Modifications to this file will cause CI to fail, the code generator tool
// must be modified to change this file.
//
// This file describes the [WASI] interface, consisting of functions, types,
// and defined values (macros).
//
// The interface described here is greatly inspired by [CloudABI]'s clean,
// thoughtfully-designed, capability-oriented, POSIX-style API.
//
// [CloudABI]: https://github.com/NuxiNL/cloudlibc
// [WASI]: https://github.com/WebAssembly/WASI/

package wasi

import (
    "github.com/pgavlin/warp/exec"
)

"#,
        inputs_str,
    ));

    let mut stubs = String::new();
    stubs.push_str("package wasi\n\n");

    let mut type_constants = HashMap::new();
    for c in doc.constants() {
        type_constants.entry(&c.ty).or_insert(Vec::new()).push(c);
    }

    for nt in doc.typenames() {
        print_datatype(&mut types, &*nt);

        if let Some(constants) = type_constants.remove(&nt.name) {
            for constant in constants {
                print_constant(&mut types, &constant);
            }
        }
    }

    for m in doc.modules() {
        print_module(&mut module, &mut stubs, &m);
    }

    Generated { module, types, stubs }
}

fn print_datatype(ret: &mut String, nt: &NamedType) {
    if !nt.docs.is_empty() {
        for line in nt.docs.lines() {
            ret.push_str(&format!("// {}\n", line));
        }
    }

    match &nt.tref {
        TypeRef::Value(v) => match &**v {
            Type::Record(s) => print_record(ret, &nt.name, s),
            Type::Variant(v) => print_variant(ret, &nt.name, v),
            Type::List(l) => print_list(ret, &nt.name, l),
            Type::Builtin { .. }
            | Type::Pointer { .. }
            | Type::ConstPointer { .. }
            | Type::Handle { .. } => print_alias(ret, &nt.name, &nt.tref),
        },
        TypeRef::Name(_) => print_alias(ret, &nt.name, &nt.tref),
    }
}

fn print_alias(ret: &mut String, name: &Id, dest: &TypeRef) {
    let type_ = match &**dest.type_() {
        Type::Builtin(b) => builtin_type_name(*b),
        Type::Pointer { .. }
        | Type::ConstPointer { .. } => "pointer",
        Type::Handle { .. } => "handle",
        _ => unreachable!(),
    };

    ret.push_str(&format!("type wasi{} = {}\n\n", &export_ident_name(name), type_));
}

fn print_enum(ret: &mut String, name: &Id, v: &Variant) {
    ret.push_str(&format!("type wasi{} = {}\n", &export_ident_name(name), &intrepr_name(v.tag_repr)));

    for (index, case) in v.cases.iter().enumerate() {
        if !case.docs.is_empty() {
            for line in case.docs.lines() {
                ret.push_str(&format!("// {}\n", line));
            }
        }
        ret.push_str(&format!("const wasi{}{} = {}\n", &export_ident_name(name), &export_ident_name(&case.name), index));
    }

    ret.push_str("\n");
}

fn print_constant(ret: &mut String, const_: &Constant) {
    if !const_.docs.is_empty() {
        for line in const_.docs.lines() {
            ret.push_str(&format!("// {}\n", line));
        }
    }
    ret.push_str(&format!("const {} = {}\n\n", ident_name(&const_.name), const_.value));
}

fn print_record(ret: &mut String, name: &Id, s: &RecordDatatype) {
    if let Some(repr) = s.bitflags_repr() {
        ret.push_str(&format!("type wasi{} = {}\n", &export_ident_name(name), &intrepr_name(repr)));
        for (i, member) in s.members.iter().enumerate() {
            if !member.docs.is_empty() {
                for line in member.docs.lines() {
                    ret.push_str(&format!("// {}\n", line));
                }
            }
            ret.push_str(&format!("const wasi{}{} = 1 << {}\n", &export_ident_name(name), &export_ident_name(&member.name), i));
        }
        ret.push_str("\n");
        return;
    }

    ret.push_str(&format!("type wasi{} struct {{\n", &export_ident_name(name)));

    for member in s.members.iter() {
        if !member.docs.is_empty() {
            for line in member.docs.lines() {
                ret.push_str(&format!("\t// {}\n", line));
            }
        }
        ret.push_str(&format!("\t{} {}\n", &ident_name(&member.name), &typeref_name(&member.tref)));
    }

    ret.push_str(&format!(r#"}}

func (v *wasi{name}) layout() (uint32, uint32) {{
    return {size}, {align}
}}

func (v *wasi{name}) store(mem *exec.Memory, addr, offset uint32) {{
    base := addr + offset
"#, name=&export_ident_name(name), size=s.mem_size(), align=s.mem_align()));

    for layout in s.member_layout().iter() {
        let member = &layout.member;
        print_store(ret, &member.tref, "mem", &format!("v.{}", &ident_name(&member.name)), "base", layout.offset);
    }

    ret.push_str(&format!(r#"}}

func (v *wasi{name}) load(mem *exec.Memory, addr, offset uint32) {{
    base := addr + offset
"#, name=&export_ident_name(name)));

    for layout in s.member_layout().iter() {
        let member = &layout.member;
        print_load(ret, &member.tref, "mem", &format!("v.{}", &ident_name(&member.name)), "base", layout.offset);
    }

    ret.push_str("}\n\n");
}

fn print_variant(ret: &mut String, name: &Id, v: &Variant) {
    if v.is_enum() {
        return print_enum(ret, name, v);
    }

    ret.push_str(&format!("type wasi{} struct {{\n", export_ident_name(name)));
    ret.push_str(&format!("\ttag {}\n\n", intrepr_name(v.tag_repr)));

    for case in &v.cases {
        if let Some(tref) = &case.tref {
            if !case.docs.is_empty() {
                for line in case.docs.lines() {
                    ret.push_str(&format!("\t// {}\n", line));
                }
            }
            ret.push_str(&format!("\t{} {}\n", ident_name(&case.name), typeref_name(tref)));
        }
    }

    ret.push_str(&format!(r#"}}

func (v *wasi{name}) layout() (uint32, uint32) {{
    return {size}, {align}
}}

func (v *wasi{name}) store(mem *exec.Memory, addr, offset uint32) {{
    base := addr + offset
"#, name=&export_ident_name(name), size=v.mem_size(), align=v.mem_align()));
    print_sized_store(ret, v.tag_repr.mem_size(), "mem", "v.tag", "base", 0);
    ret.push_str("\tswitch v.tag {\n");

    let tag_size = v.tag_repr.mem_size();
    for (i, case) in v.cases.iter().enumerate() {
        ret.push_str(&format!("\tcase {}:\n", i));

        if let Some(payload) = &case.tref {
            let offset = align_to(tag_size, payload.mem_align());
            print_store(ret, &payload, "mem", &format!("v.{}", &ident_name(&case.name)), "base", offset);
        }
    }

    ret.push_str(&format!(r#"   }}
}}

func (v *wasi{name}) load(mem *exec.Memory, addr, offset uint32) {{
    base := addr + offset
"#, name=&export_ident_name(name)));
    print_typed_load(ret, &intrepr_name(v.tag_repr), v.tag_repr.mem_size(), "mem", "v.tag", "base", 0);
    ret.push_str("\tswitch v.tag {\n");

    let tag_size = v.tag_repr.mem_size();
    for (i, case) in v.cases.iter().enumerate() {
        ret.push_str(&format!("\tcase {}:\n", i));

        if let Some(payload) = &case.tref {
            let offset = align_to(tag_size, payload.mem_align());
            print_load(ret, &payload, "mem", &format!("v.{}", &ident_name(&case.name)), "base", offset);
        }
    }

    ret.push_str(r#"    }
}

"#);
}

fn print_list(ret: &mut String, name: &Id, element: &TypeRef) {
    let element_size_align = element.mem_size_align();
    let element_size = align_to(element_size_align.size, element_size_align.align);

    ret.push_str(&format!(r#"type wasi{name} list

func (v *wasi{name}) elementSize() uint32 {{
    return {element_size}
}}

func (l *wasi{name}) storeIndex(mem *exec.Memory, index int, value {element}) {{
    addr := uint32(l.pointer) + {element_size} * uint32(index)
"#, name=&export_ident_name(name), element=&typeref_name(element), element_size=element_size));

    print_store(ret, element, "mem", "value", "addr", 0);

    ret.push_str(&format!(r#"}}

func (l *wasi{name}) loadIndex(mem *exec.Memory, index int) {element} {{
    var value {element}
    addr := uint32(l.pointer) + uint32(index) * {element_size}
"#, name=&export_ident_name(name), element=&typeref_name(element), element_size=element_size));

    print_load(ret, element, "mem", "value", "addr", 0);

    ret.push_str(r#"    return value
}

"#);
}


fn print_module(module: &mut String, stubs: &mut String, m: &Module) {
    for line in m.docs.lines() {
        module.push_str(&format!("// {}\n", line));
        stubs.push_str(&format!("// {}\n", line));
    }

    print_module_def(module, m);
    print_module_instance(module, m);

    stubs.push_str(&format!(r#"type {}Impl struct {{
}}

"#, &ident_name(&m.name)));

    for func in m.funcs() {
        print_func_stub(stubs, &func, &m.name);
        print_func_source(module, &func, &m.name);
    }
}

fn print_module_def(module: &mut String, m: &Module) {
    module.push_str(&format!("type {}Definition int\n\n", &ident_name(&m.name)));

    module.push_str(&format!("func (def {}Definition) GetImports() []wasm.ImportEntry {{\n", &ident_name(&m.name)));
    module.push_str("\treturn []wasm.ImportEntry{\n");
    module.push_str("\t}\n");
    module.push_str("}\n\n");

    module.push_str(&format!(r#"func (def {name}Definition) Allocate(name string) (exec.AllocatedModule, error) {{
    m := allocated{exported_name}{{
        {name}: &{name}{{name: name}},
    }}
    return &m, nil
}}

"#, name=&ident_name(&m.name), exported_name=&export_ident_name(&m.name)));
}

fn print_module_instance(module: &mut String, m: &Module) {
    module.push_str(&format!(r#"type {name} struct {{
    name string
    impl *{name}Impl
"#, name=&ident_name(&m.name)));
    for import in m.imports() {
        match import.variant {
            ModuleImportVariant::Memory => {
                module.push_str(&format!("\t{} *exec.Memory\n", &ident_name(&import.name)));
            }
        }
    }
    module.push_str(&format!(r#"}}

type allocated{exported_name} struct {{
    *{name}
}}

func (m *allocated{exported_name}) Instantiate(imports exec.ImportResolver) (mod exec.Module, err error) {{
"#, name=&ident_name(&m.name), exported_name=&export_ident_name(&m.name)));

    for import in m.imports() {
        match import.variant {
            ModuleImportVariant::Memory => {
                module.push_str(&format!(r#"m.{}, err = imports.ResolveMemory("", "{}", wasm.Memory{{}})
if err != nil {{
    return nil, err
}}
"#, &ident_name(&import.name), &import.name.as_str()));
            }
        }
    }

    module.push_str(&format!(r#"
    return m.{name}, nil
}}

func (m *{name}) Name() string {{
    return m.name
}}

func (m *{name}) GetTable(name string) (*exec.Table, error) {{
	return nil, errors.New("unknown table")
}}

func (m *{name}) GetMemory(name string) (*exec.Memory, error) {{
	return nil, errors.New("unknown memory")
}}

func (m *{name}) GetGlobal(name string) (*exec.Global, error) {{
	return nil, errors.New("unknown global")
}}

func (m *{name}) GetFunction(name string) (exec.Function, error) {{
    switch name {{
"#, name=&ident_name(&m.name)));

    for (index, func) in m.funcs().enumerate() {
        module.push_str(&format!(r#"case "{}":
    return exec.NewHostFunction(m, {}, reflect.ValueOf(m.wasi{})), nil
"#, &func.name.as_str(), index, export_ident_name(&func.name)));
    }

    module.push_str(&format!(r#"default:
        return nil, errors.New("unknown function")
    }}
}}

func (m *{}) mem() *exec.Memory {{
"#, &ident_name(&m.name)));

    let memory = m.imports().find(|i| match i.variant {
        ModuleImportVariant::Memory => true
    });
    match memory {
        Some(import) => module.push_str(&format!("\treturn m.{}\n", &ident_name(&import.name))),
        None => module.push_str("\treturn nil\n"),
    }
    module.push_str("}\n\n");
}

fn print_func_stub(ret: &mut String, func: &InterfaceFunc, module_name: &Id) {
    if !func.docs.is_empty() {
        for line in func.docs.lines() {
            ret.push_str(&format!("// {}\n", line));
        }
    }

    ret.push_str(&format!("func (m *{}Impl) {}(", ident_name(module_name), ident_name(&func.name)));
    for (i, param) in func.params.iter().enumerate() {
        if i > 0 {
            ret.push_str(", ");
        }
        ret.push_str(&format!("p{} {}", &ident_name(&param.name), &typeref_name(&param.tref)));
    }
    ret.push_str(")");

    match func.results.len() {
        0 => {}
        1 => {
            let v = match &**func.results[0].tref.type_() {
                Type::Variant(v) => v,
                _ => unreachable!(),
            };
            if !v.is_enum() {
                let (ok, err_option) = v.as_expected().unwrap();
                let err = err_option.unwrap();

                ret.push_str(" (");
                match ok {
                    Some(ok) => {
                        if let Some(tuple) = as_tuple(ok) {
                            for (i, m) in tuple.members.iter().enumerate() {
                                ret.push_str(&format!("r{} {}, ", i, &typeref_name(&m.tref)));
                            }
                        } else {
                            ret.push_str(&format!("rv {}, ", &typeref_name(ok)));
                        }

                        ret.push_str(&format!("err {}", &typeref_name(err)));
                    }
                    None => ret.push_str(&format!("err {}", &typeref_name(&err))),
                }
            } else {
                ret.push_str(&format!("rv {}", &typeref_name(&func.results[0].tref)));
            }
            ret.push_str(")");
        }
        _ => panic!("unsupported number of return values"),
    }

    ret.push_str(r#" {
    return
}

"#);
}

fn print_func_source(ret: &mut String, func: &InterfaceFunc, module_name: &Id) {
    if !func.docs.is_empty() {
        for line in func.docs.lines() {
            ret.push_str(&format!("// {}\n", line));
        }
    }

    let (params, results) = func.wasm_signature();

    ret.push_str(&format!("func (m *{}) wasi{}(", ident_name(module_name), export_ident_name(&func.name)));
    for (i, param) in params.iter().enumerate() {
        if i > 0 {
            ret.push_str(", ");
        }
        ret.push_str(&format!("p{} ", i));
        ret.push_str(wasm_type(param));
    }
    ret.push_str(")");

    match results.len() {
        0 => {}
        1 => ret.push_str(&format!("({})", wasm_type(&results[0]))),
        _ => panic!("unsupported number of return values"),
    }

    ret.push_str(" {\n");

    func.call_interface(
        module_name,
        &mut Go {
            src: ret,
            variants: Vec::new(),
            block_storage: Vec::new(),
            blocks: Vec::new(),
            ip: 0,
        },
    );

    ret.push_str("}\n\n");

    /// This is a structure which implements the glue necessary to translate
    /// between the Go API of a function and the desired WASI ABI we're being
    /// told it has.
    ///
    /// It's worth nothing that this will, in the long run, get much fancier.
    /// For now this is extremely simple and entirely assumes that the WASI ABI
    /// matches our Go ABI. This means it will only really generate valid code
    /// as-is *today* and won't work for any updates to the WASI ABI in the
    /// future.
    ///
    /// It's hoped that this situation will improve as interface types and witx
    /// continue to evolve and there's a more clear path forward for how to
    /// translate an interface types signature to a Go API.
    struct Go<'a> {
        src: &'a mut String,
        variants: Vec<(TypeRef, String)>,
        block_storage: Vec<String>,
        blocks: Vec<(String, Option<(Option<TypeRef>, String)>)>,
        ip: i32,
    }

    impl Bindgen for Go<'_> {
        type Operand = (Option<TypeRef>, String);

        fn push_block(&mut self) {
            let prev = mem::replace(self.src, String::new());
            self.block_storage.push(prev);
        }

        fn finish_block(&mut self, operand: Option<(Option<TypeRef>, String)>) {
            let to_restore = self.block_storage.pop().unwrap();
            let src = mem::replace(self.src, to_restore);
            self.blocks.push((src, operand));
        }

        fn allocate_space(&mut self, _: usize, _: &NamedType) {
            // not necessary due to us taking parameters as pointers
        }

        fn emit(
            &mut self,
            inst: &Instruction<'_>,
            operands: &mut Vec<(Option<TypeRef>, String)>,
            results: &mut Vec<(Option<TypeRef>, String)>,
        ) {
            self.ip += 1;

            let mut top_as = |cvt: &str| {
                results.push((None, format!("{}({})", cvt, operands.pop().unwrap().1)));
            };

            match inst {
                Instruction::GetArg { nth } => {
                    results.push((None, format!("p{}", nth)));
                }

                Instruction::I32FromPointer
                | Instruction::I32FromConstPointer
                | Instruction::I32FromHandle { .. }
                | Instruction::I32FromUsize
                | Instruction::I32FromChar
                | Instruction::I32FromU8
                | Instruction::I32FromS8
                | Instruction::I32FromChar8
                | Instruction::I32FromU16
                | Instruction::I32FromS16
                | Instruction::I32FromU32
                | Instruction::I32FromBitflags { .. } => top_as("int32"),

                Instruction::I64FromU64
                | Instruction::I64FromBitflags { .. } => top_as("int64"),

                // No conversion necessary
                Instruction::F32FromIf32
                | Instruction::F64FromIf64
                | Instruction::If32FromF32
                | Instruction::If64FromF64
                | Instruction::I32FromS32
                | Instruction::I64FromS64
                | Instruction::S32FromI32
                | Instruction::S64FromI64 => results.push(operands.pop().unwrap()),

                Instruction::S8FromI32 => top_as("int8"),
                Instruction::Char8FromI32 => top_as("byte"),
                Instruction::U8FromI32 => top_as("uint8"),
                Instruction::S16FromI32 => top_as("int16"),
                Instruction::U16FromI32 => top_as("uint16"),
                Instruction::UsizeFromI32
                | Instruction::U32FromI32 => top_as("uint32"),
                Instruction::U64FromI64 => top_as("uint64"),
                Instruction::CharFromI32 => top_as("rune"),

                Instruction::PointerFromI32 { .. }
                | Instruction::ConstPointerFromI32 { .. } => top_as("pointer"),

                Instruction::HandleFromI32 { ty }
                | Instruction::BitflagsFromI32 { ty }
                | Instruction::BitflagsFromI64 { ty } => top_as(&namedtype_name(ty)),

                Instruction::ListFromPointerLength { .. } => {
                    let (_, pointer) = operands.pop().unwrap();
                    let (_, length) = operands.pop().unwrap();
                    results.push((None, format!("list{{pointer: pointer({}), length: int32({})}}", length, pointer)));
                }

                Instruction::Store { ty } => {
                    let (_, pointer) = operands.pop().unwrap();
                    let (_, value) = operands.pop().unwrap();
                    print_named_store(self.src, &ty, "m.mem()", &value, &pointer, 0);
                }

                Instruction::TupleLower { amt } => {
                    let (ty, tuple) = operands.pop().unwrap();
                    match &**ty.unwrap().type_() {
                        Type::Record(record) => {
                            for i in 0..*amt {
                                results.push((None, tuple.clone() + &ident_name(&record.members[i].name)));
                            }
                        },
                        _ => unreachable!(),
                    };
                }

                Instruction::VariantPayload => {
                    let (type_, s) = self.variants.pop().unwrap();
                    results.push((Some(type_.clone()), s));
                }

                Instruction::ResultLower { .. } => {
                    let (_, discriminant) = operands.pop().unwrap();
                    let (err_block, err_expr) = self.blocks.pop().unwrap();
                    let (ok_block, _ok_expr) = self.blocks.pop().unwrap();

                    self.src.push_str("res := int32(wasiErrnoSuccess)\n");
                    if !ok_block.is_empty() {
                        self.src.push_str(&format!("if {} == wasiErrnoSuccess {{\n", discriminant));
                        self.src.push_str(&ok_block);
                        self.src.push_str("} else {\n");
                    } else {
                        self.src.push_str(&format!("if {} != wasiErrnoSuccess {{\n", discriminant));
                    }
                    self.src.push_str(&err_block);
                    if let Some((_, expr)) = err_expr {
                        self.src.push_str(&format!("\tres = {}\n", expr));
                    }
                    self.src.push_str("}\n");
                    results.push((None, "res".to_string()));
                }

                // Enums are represented in Go simply as the integral tag type
                Instruction::EnumLift { ty } => match &**ty.type_() {
                    Type::Variant(v) => top_as(intrepr_name(v.tag_repr)),
                    _ => unreachable!(),
                },
                Instruction::EnumLower { .. } => top_as("int32"),

                Instruction::CallInterface {
                    module: _,
                    func,
                } => {
                    assert!(func.results.len() < 2);

                    self.src.push_str("\t");
                    if func.results.len() > 0 {
                        let v = match &**func.results[0].tref.type_() {
                            Type::Variant(v) => v,
                            _ => unreachable!(),
                        };
                        if !v.is_enum() {
                            let (ok, err_option) = v.as_expected().unwrap();
                            let err = err_option.unwrap();

                            self.variants.push((err.clone(), "err".to_string()));
                            if let Some(ok) = ok {
                                if let Some(tuple) = as_tuple(ok) {
                                    for m in tuple.members.iter() {
                                        self.src.push_str(&format!("rv{}, ", &ident_name(&m.name)));
                                    }
                                } else {
                                    self.src.push_str("rv, ");
                                }
                                self.variants.push((ok.clone(), "rv".to_string()));
                            }

                            self.src.push_str("err := ");
                            results.push((Some(err.clone()), "err".to_string()));
                        } else {
                            self.src.push_str("rv := ");
                            results.push((None, "rv".to_string()));
                        }
                    }
                    self.src.push_str("m.impl.");
                    self.src.push_str(&ident_name(&func.name));
                    self.src.push_str("(");
                    for (i, (_, s)) in operands.iter().enumerate() {
                        if i > 0 {
                            self.src.push_str(", ");
                        }
                        self.src.push_str(s);
                    }
                    self.src.push_str(")\n");
                }

                Instruction::Return { amt: 0 } => {}
                Instruction::Return { amt: 1 } => {
                    self.src.push_str("\treturn ");
                    self.src.push_str(&operands[0].1);
                    self.src.push_str("\n");
                }

                other => panic!("unimplemented instruction {:?}", other),
            }
        }
    }
}

fn mem_size(ty: &Type) -> Option<usize> {
    match ty {
        Type::Record(s) => match s.bitflags_repr() {
            Some(repr) => Some(repr.mem_size()),
            None => None,
        },
        Type::Variant(s) => match s.is_enum() {
            true => Some(s.tag_repr.mem_size()),
            false => None,
        },
        Type::Handle(h) => Some(h.mem_size()),
        Type::List { .. } => Some(8), // Pointer and Length
        Type::Pointer { .. } | Type::ConstPointer { .. } => Some(BuiltinType::S32.mem_size()),
        Type::Builtin(b) => Some(b.mem_size()),
    }
}

fn print_store(ret: &mut String, ty: &TypeRef, mem: &str, value: &str, pointer: &str, offset: usize) {
    match ty {
        TypeRef::Name(ty) => print_named_store(ret, &*ty, mem, value, pointer, offset),
        TypeRef::Value(ty) => match mem_size(&*ty) {
            Some(size) => print_sized_store(ret, size, mem, value, pointer, offset),
            None => unreachable!(),
        }
    }
}

fn print_named_store(ret: &mut String, ty: &NamedType, mem: &str, value: &str, pointer: &str, offset: usize) {
    match mem_size(&**ty.type_()) {
        None => ret.push_str(&format!("{}.store({}, uint32({}), {})\n", value, mem, pointer, offset)),
        Some(size) => print_sized_store(ret, size, mem, value, pointer, offset),
    };
}

fn print_sized_store(ret: &mut String, size: usize, mem: &str, value: &str, pointer: &str, offset: usize) {
    match size {
        1 => ret.push_str(&format!("{}.PutByte(byte({}), uint32({}), {})\n", mem, value, pointer, offset)),
        2 => ret.push_str(&format!("{}.PutUint16(uint16({}), uint32({}), {})\n", mem, value, pointer, offset)),
        4 => ret.push_str(&format!("{}.PutUint32(uint32({}), uint32({}), {})\n", mem, value, pointer, offset)),
        8 => ret.push_str(&format!("{}.PutUint64(uint64({}), uint32({}), {})\n", mem, value, pointer, offset)),
        _ => unreachable!(),
    };
}

fn print_load(ret: &mut String, ty: &TypeRef, mem: &str, lvalue: &str, pointer: &str, offset: usize) {
    match ty {
        TypeRef::Name(ty) => print_named_load(ret, &*ty, mem, lvalue, pointer, offset),
        TypeRef::Value(ty) => match mem_size(&*ty) {
            Some(size) => print_typed_load(ret, &type_name(&*ty), size, mem, lvalue, pointer, offset),
            None => unreachable!(),
        }
    }
}

fn print_named_load(ret: &mut String, ty: &NamedType, mem: &str, lvalue: &str, pointer: &str, offset: usize) {
    match mem_size(&**ty.type_()) {
        None => ret.push_str(&format!("{}.load({}, uint32({}), {})\n", lvalue, mem, pointer, offset)),
        Some(size) => print_typed_load(ret, &namedtype_name(ty), size, mem, lvalue, pointer, offset),
    };
}

fn print_typed_load(ret: &mut String, lvalue_type: &str, size: usize, mem: &str, lvalue: &str, pointer: &str, offset: usize) {
    match size {
        1 => ret.push_str(&format!("{} = {}({}.Byte(uint32({}), {}))\n", lvalue, lvalue_type, mem, pointer, offset)),
        2 => ret.push_str(&format!("{} = {}({}.Uint16(uint32({}), {}))\n", lvalue, lvalue_type, mem, pointer, offset)),
        4 => ret.push_str(&format!("{} = {}({}.Uint32(uint32({}), {}))\n", lvalue, lvalue_type, mem, pointer, offset)),
        8 => ret.push_str(&format!("{} = {}({}.Uint64(uint32({}), {}))\n", lvalue, lvalue_type, mem, pointer, offset)),
        _ => unreachable!(),
    };
}

/// If the next free byte in the struct is `offs`, and the next
/// element has alignment `alignment`, determine the offset at
/// which to place that element.
fn align_to(offs: usize, alignment: usize) -> usize {
    offs + alignment - 1 - ((offs + alignment - 1) % alignment)
}

fn ident_name(i: &Id) -> String {
    let s = i.as_str();
    match s {
        "type" => "type_".to_string(),
        _ => s.to_mixed_case(),
    }
}

fn export_ident_name(i: &Id) -> String {
    i.as_str().to_camel_case()
}

fn builtin_type_name(b: BuiltinType) -> &'static str {
    match b {
        BuiltinType::U8 { lang_c_char: true } => {
            panic!("no type name for string or char8 builtins")
        }
        BuiltinType::U8 { lang_c_char: false } => "uint8",
        BuiltinType::U16 => "uint16",
        BuiltinType::U32 { .. } => "uint32",
        BuiltinType::U64 => "uint64",
        BuiltinType::S8 => "int8",
        BuiltinType::S16 => "int16",
        BuiltinType::S32 => "int32",
        BuiltinType::S64 => "int64",
        BuiltinType::F32 => "float32",
        BuiltinType::F64 => "float64",
        BuiltinType::Char => "rune",
    }
}

fn as_tuple(tref: &TypeRef) -> Option<&RecordDatatype> {
    match tref {
        TypeRef::Value(type_) => match &**type_ {
            Type::Record(record) => {
                if record.is_tuple() {
                    Some(record)
                } else {
                    None
                }
            }
            _ => None
        }
        _ => None
    }
}

fn typeref_name(tref: &TypeRef) -> String {
    match tref {
        TypeRef::Name(named_type) => namedtype_name(&named_type),
        TypeRef::Value(anon_type) => type_name(&**anon_type),
    }
}

fn type_name(type_: &Type) -> String {
    match type_ {
        Type::List(_) => "list".to_string(),
        Type::Builtin(b) => builtin_type_name(*b).to_string(),
        Type::Pointer(_)
        | Type::ConstPointer(_) => "pointer".to_string(),
        Type::Handle { .. } => "handle".to_string(),
        Type::Variant { .. } => "variant".to_string(),
        Type::Record { .. } => "record".to_string(),
    }
}

fn namedtype_name(named_type: &NamedType) -> String {
    match &**named_type.type_() {
        Type::Pointer(_)
        | Type::ConstPointer(_) => "pointer".to_string(),
        Type::List(_) => "list".to_string(),
        _ => format!("wasi{}", named_type.name.as_str().to_camel_case()),
    }
}

fn intrepr_name(i: IntRepr) -> &'static str {
    match i {
        IntRepr::U8 => "uint8",
        IntRepr::U16 => "uint16",
        IntRepr::U32 => "uint32",
        IntRepr::U64 => "uint64",
    }
}

fn wasm_type(wasm: &WasmType) -> &'static str {
    match wasm {
        WasmType::I32 => "int32",
        WasmType::I64 => "int64",
        WasmType::F32 => "float32",
        WasmType::F64 => "float64",
    }
}
