<html>
<head>
<script type="text/javascript" src="https://www.google.com/jsapi"></script>
<script type="text/javascript" src="//ajax.googleapis.com/ajax/libs/jquery/1.10.2/jquery.min.js"></script>
<script type="text/javascript">
$('document').ready(function() {
	localStorage.setItem("keystoneURL", {{.}});

$("#loginform").submit(function(event){
	url = {{.}} + '/v3/auth/tokens';
	console.log(url);
	str = '{"auth": { "identity": { "password": { "user": { "domain": { "id": "default" }, "password": "", "name": "" } }, "methods": ["password"] } } }';
	authObj = JSON.parse(str);

	console.log("form ajax submit");
	var username = document.getElementById('username').value;
	var password = document.getElementById('password').value;
	authObj.auth.identity.password.user.password = password;
	authObj.auth.identity.password.user.name = username;

	request = $.ajax({
		url: url,
		type: 'POST',
		data: JSON.stringify(authObj),
		dataType: 'json',
		contentType: 'application/json',
	})
	.done(function (data, textStatus, request) {
		var jsonObj = JSON.parse(request.responseText);
		console.log(JSON.stringify(jsonObj));
		console.log(request.getResponseHeader('X-Subject-Token'));
		localStorage.setItem("tokenObj", request.responseText);
		localStorage.setItem("auth_token", request.getResponseHeader('X-Subject-Token'));
		window.location = '/tenantDebug';
	})
	.fail(function (jqXHR, textStatus, errorThrown) {
		alert(textStatus);
	});

	event.preventDefault();
});
});
</script>
<style type="text/css">
.section-style-2 {
	width: 350px;
	float: left;
	padding: 10px;
	border-left: 1px solid;
}

progress[value] {
	-webkit-appearance: none;
	appearance: none;
	width: 250px;
	height: 20px;
    	margin: 10px 20px auto;
}

progress[value]::-webkit-progress-bar {
	background-color: #eee;
	border-radius: 2px;
	box-shadow: 0 2px 5px rgba(0, 0, 0, 0.25) inset;
}

progress[value]::-webkit-progress-value {
	background-image:
		-webkit-linear-gradient(-45deg,
				transparent 33%, rgba(0, 0, 0, .1) 33%,
				rgba(0, 0, 0, .1) 66%, transparent 66%),
		-webkit-linear-gradient(top,
				rgba(255, 255, 255, .25),
				rgba(0, 0, 0, .25)),
		-webkit-linear-gradient(left, #4B99AD, #f44);
	border-radius: 2px;
	background-size: 35px 20px, 100% 100%, 100% 100%;
}

.form-style-1 {
    margin:10px auto;
    max-width: 400px;
    padding: 20px 12px 10px 20px;
    font: 13px "Lucida Sans Unicode", "Lucida Grande", sans-serif;
}

.form-style-1 li {
    padding: 0;
    display: block;
    list-style: none;
    margin: 10px 0 0 0;
}
.form-style-1 label{
    margin:0 0 3px 0;
    padding:0px;
    display:block;
    font-weight: bold;
}
.form-style-1 input[type=text], 
textarea, 
select{
    box-sizing: border-box;
    -webkit-box-sizing: border-box;
    -moz-box-sizing: border-box;
    border:1px solid #BEBEBE;
    padding: 7px;
    margin:0px;
    -webkit-transition: all 0.30s ease-in-out;
    -moz-transition: all 0.30s ease-in-out;
    -ms-transition: all 0.30s ease-in-out;
    -o-transition: all 0.30s ease-in-out;
    outline: none;  
}
.form-style-1 input[type=text]:focus, 
.form-style-1 input[type=submit], .form-style-1 input[type=button]{
    background: #4B99AD;
    padding: 8px 15px 8px 15px;
    border: none;
    color: #fff;
}

.form-style-1 input[type=submit]:hover, .form-style-1 input[type=button]:hover{
    background: #4691A4;
    box-shadow:none;
    -moz-box-shadow:none;
    -webkit-box-shadow:none;
}
</style>
</head>
<body>
</head>
<body>
<section>
<form name="login" id="loginform">
<ul class="form-style-1"><label>Login</label>
	<li><label>User Name</label>
		<input type='text' name='username' id='username' maxlength="50"></input>
	</li>
	<li><label>Password</label>
		<input type='password' name='password' id='password' maxlength="50"></input>
	</li>
	<li>
		<input type='submit' name='Submit' value='Submit'></input>
	</li>
</ul>
</form>
</section>
</body>
</html>
