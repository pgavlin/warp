#[macro_use]
extern crate clap;

use anyhow::Result;
use clap::{Arg, SubCommand};
use std::fs;
use std::path::Path;
use gen_wasi::{generate, wasi_api_module, wasi_api_types, wasi_api_stubs, snapshot_witx_files};

pub fn run(
    inputs: &[impl AsRef<Path>],
    module: impl AsRef<Path>,
    types: impl AsRef<Path>,
    stubs: impl AsRef<Path>,
) -> Result<()> {
    let go_defs = generate(inputs)?;
    fs::write(module, go_defs.module)?;
    fs::write(types, go_defs.types)?;
    fs::write(stubs, go_defs.stubs)?;
    Ok(())
}

fn main() -> Result<()> {
    let matches = app_from_crate!()
        .setting(clap::AppSettings::SubcommandRequiredElseHelp)
        .subcommand(
            SubCommand::with_name("generate")
                .arg(Arg::with_name("inputs").required(true).multiple(true))
                .arg(
                    Arg::with_name("output")
                        .short("o")
                        .long("output")
                        .takes_value(true)
                        .required(true),
                ),
        )
        .subcommand(
            SubCommand::with_name("generate-api")
                .about("generate WASI API from current snapshot"),
        )
        .get_matches();

    if matches.subcommand_matches("generate-api").is_some() {
        let inputs = snapshot_witx_files()?;
        run(&inputs, wasi_api_module(), wasi_api_types(), wasi_api_stubs())?;
    } else if let Some(generate) = matches.subcommand_matches("generate") {
        let inputs = generate
            .values_of("inputs")
            .expect("required inputs arg")
            .collect::<Vec<_>>();
        let output = generate.value_of("output").expect("required output arg");
        let output = Path::new(&output);
        run(
            &inputs,
            output.with_extension("module.go"),
            output.with_extension("types.go"),
            output.with_extension("stubs.go"),
        )?;
    } else {
        unreachable!("a subcommand must be provided")
    };

    Ok(())
}
