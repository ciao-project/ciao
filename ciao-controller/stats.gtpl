<html>
<head>
<style>
.stats-box {
	display: inline-block;
	width: 400px;
	height: 200px;
	border: 3px solid blue;
}

.instances-box {
	border: 3px solid red;
}
</style>
	<!--Load the AJAX API-->
<script type="text/javascript" src="https://www.google.com/jsapi"></script>
<script type="text/javascript" src="//ajax.googleapis.com/ajax/libs/jquery/1.10.2/jquery.min.js"></script>
<script type="text/javascript">
google.load('visualization', '1.0', {'packages':['corechart', 'table',  'line']});
google.setOnLoadCallback(drawChart);

function processFrameStats (jsonData) {
	console.log("processFrameStats")
	var data = new google.visualization.DataTable();
	var stats = JSON.parse(jsonData.responseText);

	data.addColumn('string', 'Batch Label')
	data.addColumn('number', 'Number of Instances')

	for (var i in stats) {
		data.addRow([stats[i].batch_id, stats[i].num_instances]);
	}

	return data;
}

function processCNCI (jsonData) {
	console.log("processCNCI");
	var data = new google.visualization.DataTable();
	var cncis = JSON.parse(jsonData.responseText);

	data.addColumn('string', 'Tenant ID');
	data.addColumn('string', 'IP Address');
	data.addColumn('string', 'MAC Address');
	data.addColumn('string', 'Instance ID');

	for (var i in cncis) {
		data.addRow([cncis[i].tenant_id, cncis[i].ip_address, cncis[i].mac_address, cncis[i].instance_id]);
	}

	return data;
}

function processNodeSummary (jsonData) {
	console.log("processNodeSummary");
	var data = new google.visualization.DataTable();
	var jsonObj = JSON.parse(jsonData.responseText);

	data.addColumn('string', 'node_id');
	data.addColumn('number', 'Total Instances');
	data.addColumn('number', 'Total Running');
	data.addColumn('number', 'Total Pending');
	data.addColumn('number', 'Total Paused');

	$.each(jsonObj, function(index, item) {
		data.addRow([item.node_id, item.total_instances, item.total_running_instances, item.total_pending_instances, item.total_paused_instances]);
	});

	function getSum(data, column) {
		var total = 0;
		for (var i = 0; i < data.getNumberOfRows(); i++) {
			total += data.getValue(i, column);
		}
		return total;
	}

	data.addRow(["Total", getSum(data, 1), getSum(data, 2), getSum(data, 3), getSum(data, 4)]);

	return data;
}

function processEventLog (jsonData) {
	console.log("processEventLog");
	var data = new google.visualization.DataTable();
	var jsonObj = JSON.parse(jsonData.responseText);

	data.addColumn('datetime', 'timestamp');
	data.addColumn('string', 'type');
	data.addColumn('string', 'message');
	data.addColumn('string', 'tenant_id');

	$.each(jsonObj, function(index, item) {
		var date = new Date(item.time_stamp);
		data.addRow([date, item.type, item.message, item.tenant_id]);
	});
	return data;
}

function processInstanceStats (jsonData) {
	console.log("processInstanceStats");
	var data = new google.visualization.DataTable();
	var jsonObj = JSON.parse(jsonData.responseText);

	data.addColumn('string', 'instance_id');
	data.addColumn('string', 'tenant_id');
	data.addColumn('string', 'state');
	data.addColumn('string', 'workload_id');
	data.addColumn('string', 'ip_address');
	data.addColumn('string', 'mac_address');
	data.addColumn('string', 'node_id');

	$.each(jsonObj, function(index, item) {
		data.addRow([item.instance_id, item.tenant_id, item.instance_state, item.workload_id, item.ip_address, item.mac_address, item.node_id]);
	});
	return data;
}

function processNodeStats (jsonData) {
	console.log("processNodeStats");
	var data = new google.visualization.DataTable();

	data.addColumn('datetime', 'Time Stamp');
	data.addColumn('string', 'Node ID');
	data.addColumn('number', 'Load');
	data.addColumn('number', 'Mem Available (MB)');
	data.addColumn('number', 'Disk Available (MB)');
	data.addColumn('number', 'CPUs Online');

	if (jsonData.responseText.length > 0) {
		var jsonObj = JSON.parse(jsonData.responseText);
		$.each(jsonObj, function(index, item) {
			var date = new Date(item.time_stamp);
			data.addRow([date, item.node_id, item.load, item.mem_available_mb, item.disk_available_mb, item.cpus_online]);
		});
	}
	return data;
}

function makeChartData(data, column_index) {
	var view = new google.visualization.DataView(data);
	var node_ids = view.getDistinctValues(1);
	var combinedView = new google.visualization.DataTable();

	// for each node, make a view of stats
	for (var i = 0; i < node_ids.length; i++) {
		var v = new google.visualization.DataView(data);
		var node_filter = data.getFilteredRows([{
			column: 1,
			value: node_ids[i]
		}]);
		v.setRows(node_filter);
		v.setColumns([0, column_index]);
		var t = v.toDataTable();
		t.setColumnLabel(1, node_ids[i]);
		if (i == 0) {
			combinedView = t;
		} else {
			var numCols = combinedView.getNumberOfColumns();
			var cols = [];
			for (var j = 1; j < numCols; j++) {
				cols.push(j);
			}
			combinedView = google.visualization.data.join(combinedView, t, 'full', [[0,0]], cols, [1]);
		}
	}

	return combinedView;
}

function drawChart() {
	var tableData;
	var nodeData;
	var eventLogData;
	var frameData;

	var loadOptions = {
		'title':'Node Load',
		chartArea: {width: "80%", height: "70%"},
		'interpolateNulls':true,
		'legend': 'none',
	};
	var memOptions = {
		'title':'Node Available Mem (MB)',
		chartArea: {width: "70%", height: "70%"},
		'interpolateNulls':true,
		'legend': 'none',
	};
	var diskOptions = {
		'title':'Node Available Disk (MB)',
		chartArea: {width: "70%", height: "70%"},
		'interpolateNulls':true,
		'legend': 'none',
	};

	var loadChart = new google.visualization.LineChart(
				document.getElementById('load_div'));
	var memChart = new google.visualization.LineChart(
				document.getElementById('mem_div'));
	var diskChart = new google.visualization.LineChart(
				document.getElementById('disk_div'));
	var table = new google.visualization.Table(
				document.getElementById('table_div'));
	var events = new google.visualization.Table(
				document.getElementById('events_div'));
	var summary = new google.visualization.Table(
				document.getElementById('summary_div'));
	var actions = document.getElementById('actions_div');
	var cncis = new google.visualization.Table(
				document.getElementById('cncis_div'));
	var frameChart = new google.visualization.Table(document.getElementById('frames_div'));
	var tracing = document.getElementById('tracing_action');

	function updateChart () {
		var jsonData = $.ajax({
			url: "/getNodeStats",
			dataType: "json",
		})
		.done(function (data, textStatus, jsonData) {
			var chartData = processNodeStats(jsonData);
			var loadChartData = makeChartData(chartData, 2);
			var memChartData = makeChartData(chartData, 3);
			var diskChartData = makeChartData(chartData, 4);
			loadChart.draw(loadChartData, loadOptions);
			memChart.draw(memChartData, memOptions);
			diskChart.draw(diskChartData, diskOptions);
			setTimeout(updateChart, 10000);
		}) ;
	}

	function showNodeActivity () {
		var jsonData = $.ajax({
			url: "/getInstances",
			dataType: "json"
		})
		.done(function (data, textStatus, jsonData) {

			tableData = processInstanceStats(jsonData);
			table.draw(tableData, {showRowNumber: 'true', allowHtml: 'true', width: '100%'});
		});
	}

	function showNodeSummary () {
		var jsonData = $.ajax({
			url: "/getNodeSummary",
			dataType: "json"
		})
		.done(function (data, textStatus, jsonData) {
			nodeData = processNodeSummary(jsonData);
			summary.draw(nodeData, {showRowNumber: 'true', allowHtml: 'true', width: '100%'});
		});
	}

	function showEventLog () {
		var jsonData = $.ajax({
			url: "/getEventLog",
			dataType: "json"
		})
		.done(function (data, textStatus, jsonData) {
			var logData = processEventLog(jsonData);
			events.draw(logData, {showRowNumber: 'true', allowHtml: 'true', width: '100%'});
		});
	}

	function showCNCI () {
		var jsonData = $.ajax({
			url: "/getCNCI",
			dataType: "json"
		})
		.done(function (data, textStatus, jsonData) {
			var cnciData = processCNCI(jsonData);
			cncis.draw(cnciData, {showRowNumber: 'true', allowHtml: 'true', width: '100%'});
		});
	}

	function showFrameStats () {
		var jsonData = $.ajax({
			url: "/getBatchFrameSummaryStats",
			dataType: "json"
		})
		.done(function (data, textStatus, jsonData) {
			frameData = processFrameStats(jsonData);
			frameChart.draw(frameData, {showRowNumber: 'true', allowHtml: 'true', width: '100%'});
		});
	}

	function updateStats () {
		updateChart();
		showNodeSummary();
		showNodeActivity();
		showEventLog();
		showCNCI();
		showFrameStats();
	}
	updateStats();

	tracing.onclick = function() {
		var id;
		var selectedItems = frameChart.getSelection();

		if (selectedItems.length == 0) {
			alert("you have to select a row");
			return;
		}

		var item = selectedItems[0];
		if (item.row == null) {
			return;
		}

		localStorage.setItem("batch_id", frameData.getValue(item.row, 0));
		window.location = '/framestats';
	}

	actions.onclick = function() {
		console.log("actions onclick");
		var selectedAction = document.getElementById('action').value
		console.log("action: %s", selectedAction);

		switch (selectedAction) {
		case "clearEventLog":
		case "deleteAll":
		case "cleanAll":
			break;
		case "evacuate":
			var selectedItems = summary.getSelection();
			var message = '';

			if (selectedItems.length == 0) {
				alert("Pick a Node to evacuate from the Node Summary Table");
				return;
			}

			for (var i = 0; i < selectedItems.length; i++) {
				var item = selectedItems[i];
				if (item.row != null) {
					var id = nodeData.getValue(item.row, 0);
					console.log(id);
					message += id + ' ';
				}
			}
			console.log(message);
			document.getElementById('node_ids').value = message;
			break;
		default:
			var selectedItems = table.getSelection();
			var message = '';

			if (selectedItems.length == 0) {
				alert("You have to select an item");
				return;
			}

			for (var i = 0; i < selectedItems.length; i++) {
				var item = selectedItems[i];
				if (item.row != null) {
					var status = tableData.getValue(item.row, 2);
					if (selectedAction != "clean" && status == "pending") {
						alert("I can't do that on a pending instance");
					} else {
						var id = tableData.getValue(item.row, 0);
						console.log(id);
						message += id + ' ';
					}
				}
			}
			console.log(message);
			document.getElementById('instances_ids').value = message;
		}
		document.getElementById('actions_form').submit();
	}
}
</script>
</head>
<body>
<section>
	<div id="load_div" class="stats-box"></div>
	<div id="mem_div" class="stats-box"></div>
	<div id="disk_div" class="stats-box"></div>
</section>
<section>
	<h1>Tracing</h1>
	<input type="submit" id="tracing_action" value="Show Batch Trace Details">
<section>
	<div id="frames_div"></div>
</section>
<section>
	<h1>Admin Menu</h1>
	<form action=/stats method="post" id="actions_form">
		<input type='hidden' name="instances_ids" id="instances_ids" value=''></input>
		<input type='hidden' name="node_ids" id="node_ids" value=''></input>
		<select id="action" name="admin_action">
			<option value="delete">Delete Instance</option>
			<option value="clean">Clean Up Instance</option>
			<option value="stop">Stop Instance</option>
			<option value="restart">Restart Instance</option>
			<option value="evacuate">Evacuate Node</option>
			<option value="clearEventLog">Clear Event Log</option>
			<option value="deleteAll">Delete All Instances</option>
			<option value="cleanAll">Clean Up All Instances</option>
		</select>
		<button id="actions_div">Submit</button>
	</form>
</section>
<section>
	<h1>Summary</h1>
	<div id="summary_div"></div>
</section>
<section>
	<h1>Event Log</h1>
	<div id="events_div"></div>
</section>
<section>
	<h1>Networking</h1>
	<div id="cncis_div"></div>
</section>
<section>
	<h1>Instances</h1>
	<div id="table_div"></div>
</section>
</body>
</html>
