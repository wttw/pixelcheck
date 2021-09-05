From: Whoever <{{ .From }}>
To: Me <{{ .To }}>
Date: {{ .Date }}
Subject: I'm a test email
MIME-Version: 1.0
Content-Type: text/html

<html><body>
<p>I'm a little teapot.</p>
<p><img src="{{ .Image }}"></p>
<p>Short and stout.</p>
<p>{{ .Note | html }}</p>
</body></html>