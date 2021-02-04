mod go_defs;

use anyhow::Result;
pub use go_defs::{to_go, Generated};
use std::path::{Path, PathBuf};
use witx::load;

pub fn generate<P: AsRef<Path>>(inputs: &[P]) -> Result<Generated> {
    let doc = load(&inputs)?;

    let inputs_str = &inputs
        .iter()
        .map(|p| {
            p.as_ref()
                .file_name()
                .unwrap()
                .to_str()
                .unwrap()
                .to_string()
        })
        .collect::<Vec<_>>()
        .join(", ");

    Ok(to_go(&doc, &inputs_str))
}

pub fn snapshot_witx_files() -> Result<Vec<PathBuf>> {
    witx::phases::snapshot()
}

pub fn wasi_api_module() -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("../module_definition.go")
}

pub fn wasi_api_types() -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("../types.go")
}

pub fn wasi_api_stubs() -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("../stubs.go")
}
