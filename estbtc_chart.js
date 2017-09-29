var MongoClient = require('mongodb').MongoClient;
var url = "mongodb://localhost:27017/v2";
//var logtime =[];
//var estbtc =[];
var points =[];
var http = require('http');


MongoClient.connect(url, function(err, db) {
    if (err) throw err;
    db.collection("LogEstBTC").find({}).toArray(function(err, result) {
        if (err) throw err;
        result.forEach(function(e){
            var point = {x:e.logtime, y:e.estbtc};
            points.push(point);
            //estbtc.push(e.estbtc);

        });
        db.close();

        //create a server object:
        http.createServer(function (req, res) {
            //res.write('Hello World!'); //write a response to the client
            res.write(JSON.stringify(points));
            res.end(); //end the response
        }).listen(8080); //the server object listens on port 8080

    });
});
