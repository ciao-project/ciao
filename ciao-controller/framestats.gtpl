<html>
<head>
<style>
#frames_div {
overflow-x: scroll;
overflow-y: hidden;
width: 100%;
height: 550px;
}
</style>
<script type="text/javascript" src="https://www.google.com/jsapi"></script>
<script type="text/javascript" src="//ajax.googleapis.com/ajax/libs/jquery/1.10.2/jquery.min.js"></script>
<script type="text/javascript">
google.load('visualization', '1.0', {'packages':['corechart', 'table',  'line']});
google.setOnLoadCallback(drawChart);

function processFrameStats (jsonData) {
	console.log("processFrameStats")
	var data = new google.visualization.DataTable();
	var stats = JSON.parse(jsonData.responseText);

	data.addColumn('string', 'Node ID')
	data.addColumn('number', 'Network Latency')
	data.addColumn('number', 'Controller Time')
	data.addColumn('number', 'Scheduler Time')
	data.addColumn('number', 'Launcher Time')

	for (var i in stats) {
		var total_component_time = stats[i].total_controller_time + stats[i].total_scheduler_time + stats[i].total_launcher_time;
		var network_latency = stats[i].total_elapsed_time - total_component_time;
		data.addRow([stats[i].node_id, network_latency, stats[i].total_controller_time, stats[i].total_scheduler_time, stats[i].total_launcher_time]);
	}

	return data;
}

function processSummary(jsonData) {
	var summary = JSON.parse(jsonData.responseText);
	document.getElementById("num_instances").innerHTML = summary[0].num_instances;
	document.getElementById("total_elapsed").innerHTML = summary[0].total_elapsed;
	document.getElementById("average_elapsed").innerHTML = summary[0].average_elapsed;
	document.getElementById("average_controller_elapsed").innerHTML = summary[0].average_controller_elapsed;
	document.getElementById("average_launcher_elapsed").innerHTML = summary[0].average_launcher_elapsed;
	document.getElementById("average_scheduler_elapsed").innerHTML = summary[0].average_scheduler_elapsed;
	document.getElementById("controller_variance").innerHTML = summary[0].controller_variance;
	document.getElementById("launcher_variance").innerHTML = summary[0].launcher_variance;
	document.getElementById("scheduler_variance").innerHTML = summary[0].scheduler_variance;
	document.getElementById("num_instances").innerHTML = summary[0].num_instances;
}

function drawChart() {
	var frameChart = new google.visualization.ColumnChart(document.getElementById('frames_div'));
	var frameStatsOptions = {
		'title': 'Total Elapsed Time per Component',
		isStacked: true,
		hAxis: {
			title: 'Node ID'
		},
		vAxis: {
			title: 'Elapsed Time'
		}
	};
	id = localStorage.batch_id;
	var formData = {
		'batch_id' : id
	};
	request = $.ajax({
		url: '/getBatchFrameStats',
		type: 'POST',
		data: formData,
		datatype: 'json',
	})
	.done(function (data, textStatus, request) {
		frameData = processFrameStats(request);
		frameStatsOptions.width = frameData.getNumberOfRows() * 12 + 750;
		frameStatsOptions.bar = {groupWidth: 10};
		frameChart.draw(frameData, frameStatsOptions);
	})
	.fail(function (jqXHR, textStatus, errorThrown) {
		alert(textStatus);
	});

	request = $.ajax({
		url: '/getFrameStats',
		type: 'POST',
		data: formData,
		datatype: 'json',
	})
	.done(function (data, textStatus, request) {
		processSummary(request)
	})
	.fail(function (jqXHR, textStatus, errorThrown) {
		alert(textStatus);
	});
}

</script>
</head>
<body>
<section>
	<h1>Summary</h1>
	<table>
		<tr>
			<td>Number of Instances Launched</td>
			<td id="num_instances"></td>
		</tr>
		<tr>
			<td>Total Elapsed Time to Launch All Instances</td>
			<td id="total_elapsed"></td>
		</tr>
		<tr>
			<td>Average Elapsed Time Per Instance</td>
			<td id="average_elapsed"></td>
		</tr>
		<tr>
			<td>Average Elapsed Time for Controller</td>
			<td id="average_controller_elapsed"></td>
		</tr>
		<tr>
			<td>Average Elapsed Time for Launcher</td>
			<td id="average_launcher_elapsed"></td>
		</tr>
		<tr>
			<td>Average Elapsed Time for Scheduler</td>
			<td id="average_scheduler_elapsed"></td>
		</tr>
		<tr>
			<td>Controller Variance</td>
			<td id="controller_variance"></td>
		</tr>
		<tr>
			<td>Launcher Variance</td>
			<td id="launcher_variance"></td>
		</tr>
		<tr>
			<td>Scheduler Variance</td>
			<td id="scheduler_variance"></td>
		</tr>
	</table>
	<div id="summary_div"></div>
</section>
<section>
	<h1>Component Elapsed Time per Frame</h1>
	<div id="frames_div"></div>
</section>
</body>
</html>
