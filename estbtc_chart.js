var MongoClient = require('mongodb').MongoClient;
var url = "mongodb://localhost:27017/v2";
//var logtime =[];
//var estbtc =[];

var http = require('http');




//create a server object:
http.createServer(function (req, res) {
    MongoClient.connect(url, function(err, db) {

        if (err) throw err;
        var points = "";
        db.collection("LogEstBTC").find({  }).toArray(function(err, result) {
            if (err) throw err;
            result.forEach(function(e){
                points = points + "{x:" + e.logtime.getTime() + ", y:" + e.estbtc + "},"

                //var point = {x:e.logtime, y:e.estbtc};
                //points.push(point);
                //estbtc.push(e.estbtc);
            });


            db.close();


            res.writeHead(200, {'Content-Type': 'text/html'});
            //res.write('Hello World!'); //write a response to the client
            res.write("<html>\n" +
                "   <head>\n" +
                "      <meta charset=\"UTF-8\">\n" +
                "      <script src=\"https://ajax.googleapis.com/ajax/libs/jquery/3.2.1/jquery.min.js\"></script>\n" +
                "      <script src=\"https://canvasjs.com/assets/script/canvasjs.min.js\"></script>\n" )
            res.write("  <script>\n" +

                "          $( document ).ready(function() {\n" +
                "         var chart = new CanvasJS.Chart(\"chartContainer\",\n" +
                "    {\n" +
                "      zoomEnabled: true,      \n" +
                "      \n" +
                "      title:{\n" +
                "       text: \"Est BTC during time\"       \n" +
                "     },\n" +
                "axisY:{\n" +
                "        includeZero: false\n" +
                "\n" +
                "      }," +
                "       axisX:{\n" +
                "        title: \"Day\",\n" +
                "        gridThickness: 2,\n" +
                "        interval:1, \n" +
                "        intervalType: \"day\",        \n" +
                "        valueFormatString: \"YYYY-MM-DD\", \n" +
                "        labelAngle: -20\n" +
                "      }," +

                "     data: [\n" +
                "     {        \n" +
                "      type: \"line\",\n" +

                "      xValueType: \"dateTime\",\n" +
                "      dataPoints: [                  \n" +
                points +
                "      ]\n" +
                "    }\n" +
                "    ]\n" +
                "  });\n" +
                "\n" +
                "    chart.render();    " +
                "        });" +
                "\n" +
                "      </script>\n" +
                "   </head>\n" +
                "<body>\n" +
                " <div id=\"chartContainer\" style=\"height: 80%; width: 100%;\">\n" +
                "  </div>" +
                "</body>\n" +
                "</html>"
            );





            //res.write();
            res.end(); //end the response


        });
    });


}).listen(8080); //the server object listens on port 8080

