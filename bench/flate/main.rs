use std::io::{copy,stdin,stdout};
use flate2::Compression;
use flate2::read::{GzDecoder, GzEncoder};

fn main() {
    let r = copy(&mut GzDecoder::new(GzEncoder::new(&mut stdin(), Compression::default())), &mut stdout());
    match r {
        Ok(_) => (),
        Err(e) => println!("compress/decompress pipeline failed: {}", e)
    };
}
