package handler

import (
	"fmt"
	"net/http"
	"pfg/src/server/config"
	"pfg/src/server/logs"
	"strconv"
	"time"
)

var serverStatus string

func init() {

}

// ServerStatus tries to handle the request
func ServerStatus(w http.ResponseWriter, r *http.Request) {

	r.ParseForm() //ParseForm populates r.Form and r.PostForm. (GET and POST parameters)
	GET := parseGET(r)

	if GET["data"] == "true" {

		currentTime := int32(time.Now().Round(5 * time.Second).Unix()) //Timestamp rounded to 5 seconds

		labels := " ["
		values := " ["

		for pastTime := currentTime; currentTime-pastTime < 300; pastTime -= 5 {
			labels += `"` + fmt.Sprint(pastTime) + `", `
			values += `"` + fmt.Sprint(logs.RequestStats[pastTime]) + `", `
		}

		labels += "]"
		values += "]"

		serverStatus = `
		data = {
			labels: ` + labels + `,
			datasets: [
				{
					name: "Requests", type: "bar",
					values: ` + values + `
				}
			]
		}
		chart.update(data);
		`

	} else {

		serverStatus = `<!doctype html>
		<html lang="en">
		<head>
			<meta charset="utf-8">
			<title>` + config.ServerSignature + ` status for ` + config.Address + `:` + strconv.Itoa(config.Port) + `</title>
		
			<style>
			body{
				background-color: #effcff;
				font-family: Arial, sans-serif;
			}
			.title{
			  background-color: #00b0dc;
				  color: white;
				  padding: 6px;
			}
			.logo{
				  padding-top:48px;
				  padding-bottom:12px;
			}
			.content{
			  margin: auto;
			  min-width: 480px;
			  width: 60%;
			  margin-bottom: 50px;
			}
			</style>
		</head>
		
		<body>
		
			<center>
		
			<div class="logo">
			<img src="data:image/png;base64,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">
			</div>
		
			<h1 class="title">` + config.ServerSignature + ` status for ` + config.Address + `:` + strconv.Itoa(config.Port) + `</h1>
			</center>
			<br>
		
			<div class="content">
				
				<script src="https://unpkg.com/frappe-charts@1.2.4/dist/frappe-charts.min.iife.js"></script>
				<div id="chart"></div>
				<script>
				data = {
					labels: ["empty"],
					datasets: [
						{
							name: "Requests", type: "bar",
							values: ["0"]
						}
					]
				}
				
				const chart = new frappe.Chart("#chart", {
					title: "Recent requests",
					data: data,
					type: 'line',
					height: 250,
					colors: ['#00b0dc'],
					lineOptions: {
						regionFill: 1, // default: 0
					},
					axisOptions: {
						xIsSeries: true // default: false
					},
					valuesOverPoints: 1, // default: 0
					
				})
				</script>

				<script>
				function loadScript(url, callback){

					var script = document.createElement("script")
					script.type = "text/javascript";
				
					if (script.readyState){  //IE
						script.onreadystatechange = function(){
							if (script.readyState == "loaded" ||
									script.readyState == "complete"){
								script.onreadystatechange = null;
								callback();
							}
						};
					} else {  //Others
						script.onload = function(){
							callback();
						};
					}
				
					script.src = url;
					document.getElementsByTagName("head")[0].appendChild(script);
				}
				</script>

				<script type="text/javascript">
				function updateStatus(){
					loadScript("/server-status/?data=true", function(){
						});
					setTimeout(updateStatus, 5000);
				}
				updateStatus();
				</script>

				<br>
				<center>
				<p>Current time: ` + time.Now().UTC().String() + `</p>` + `
				<p>Server Version: ` + config.ServerSignature + `</p>
				</center>
		
			</div>



		</body>
		</html>`

	}

	logs.WriteRequest(r.RemoteAddr, r.Method, r.Host, r.RequestURI, r.Proto)
	w.Header().Set("Server", config.ServerSignature)
	w.Header().Add("Content-Type", "text/html")
	fmt.Fprint(w, serverStatus)

}
