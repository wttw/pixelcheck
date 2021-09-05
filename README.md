# pixelcheck
Some simple code to help with testing MUA image load behaviour

For playing with Apple MPP behaviour. It send mail, serves images and tracks interaction.

Configuration is all in `pixelcheck.json`. It needs access to a local postgresql database, a smarthost for sending email and the ability to listen on an external address.

`pixelserve` is a webserver that listens on the configured `listen` address and serve images, recording the access in the `load` database
table. On first run it will create the needed database tables. If `tlscert` and `tlskey` are set, pointing to certificate files, it will
serve over TLS, otherwise over regular http.

`pixelsend` will send an email using the configured smarthost using a template. It will register a unique image URL in the database
and use that in the message.

The default template is `default.tpl`, this can be overridden with the `--template` flag. It's a standard Go template, with access to these variables:

  * .Date - the current date and time in RFC822Z format
  * .To - the recipient email address
  * .From - the senders email address
  * .Note - the value of the `--note` flag
  * .Image - the unique URL for the image

Usage:
```shell
pixelsend --note="testing apple" --to=whoever@example.com
```
This will create a new image, based on the default image configured in `pixelcheck.yaml`, use that and the default message template `default.tpl` to send mail to whoever@example.com. It will record the image in the `image` table and the mail sent in the `mail` table.

```shell
pixelserve
```
This will run forever, serving images. When it gets a request for a valid image it will serve the image, record everything about the request in the `load` table, and log the request to stdout.

Once some images have been loaded you can query them with, e.g.
```postgresql
select load.headers,
       mail.note,
       mail.sent,
       load.loaded,
       load.loaded-mail.sent 
from mail, load 
where mail.image=load.image
order by mail.sent;
```

to get results like
```
-[ RECORD 1 ]------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
headers  | {"Accept": ["image/webp,image/png,image/svg+xml,image/*;q=0.8,video/*;q=0.8,*/*;q=0.5"], "User-Agent": ["Mozilla/5.0"], "Accept-Encoding": ["gzip, deflate, br"], "Accept-Language": ["en-GB,en;q=0.9"]}
note     | leave unopened
sent     | 2021-09-05 11:24:10.968669+00
loaded   | 2021-09-05 12:08:04.082956+00
?column? | 00:43:53.114287
-[ RECORD 2 ]------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
headers  | {"Accept": ["image/webp,image/png,image/svg+xml,image/*;q=0.8,video/*;q=0.8,*/*;q=0.5"], "User-Agent": ["Mozilla/5.0"], "Accept-Encoding": ["gzip, deflate, br"], "Accept-Language": ["en-GB,en;q=0.9"]}
note     | fake tagged
sent     | 2021-09-05 11:24:29.06476+00
loaded   | 2021-09-05 12:08:04.082858+00
?column? | 00:43:35.018098
-[ RECORD 3 ]------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
headers  | {"Accept": ["image/webp,image/png,image/svg+xml,image/*;q=0.8,video/*;q=0.8,*/*;q=0.5"], "User-Agent": ["Mozilla/5.0"], "Accept-Encoding": ["gzip, deflate, br"], "Accept-Language": ["en-GB,en;q=0.9"]}
note     | email tagged
sent     | 2021-09-05 11:25:17.335403+00
loaded   | 2021-09-05 12:08:04.084282+00
?column? | 00:42:46.748879
-[ RECORD 4 ]------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
headers  | {"Accept": ["image/webp,image/png,image/svg+xml,image/*;q=0.8,video/*;q=0.8,*/*;q=0.5"], "User-Agent": ["Mozilla/5.0"], "Accept-Encoding": ["gzip, deflate, br"], "Accept-Language": ["en-GB,en;q=0.9"]}
note     | multipart
sent     | 2021-09-05 11:25:44.196845+00
loaded   | 2021-09-05 12:08:04.084607+00
?column? | 00:42:19.887762
```
