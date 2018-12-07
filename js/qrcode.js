var fs = require("fs");

image = require('get-image-data');
BitMatrix = require("./jsqrcode/src/bitmat.js");
require("./jsqrcode/src/grid.js");
require("./jsqrcode/src/qrcode.js");
FinderPatternFinder = require("./jsqrcode/src/findpat.js");
Detector = require("./jsqrcode/src/detector.js");
GF256Poly = require("./jsqrcode/src/gf256poly.js");
GF256 = require("./jsqrcode/src/gf256.js");
ReedSolomonDecoder = require("./jsqrcode/src/rsdecoder.js");
Decoder = require("./jsqrcode/src/decoder.js");
Version = require("./jsqrcode/src/version.js");
FormatInformation = require("./jsqrcode/src/formatinf.js");
ErrorCorrectionLevel = require("./jsqrcode/src/errorlevel.js");
DataBlock = require("./jsqrcode/src/datablock.js");
BitMatrixParser = require("./jsqrcode/src/bmparser.js");
require("./jsqrcode/src/datamask.js");
AlignmentPatternFinder = require("./jsqrcode/src/alignpat.js");
QRCodeDataBlockReader = require("./jsqrcode/src/databr.js");


fs.readFile(process.argv[2], function(err, file) {
    try{
        qrcode.decode(file, function(status,result){
            if(result !== null){
                console.log(result);
            }
        });
    } catch(e) {
        
    }
});
            