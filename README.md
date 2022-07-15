# Status Alert

A simple http request checker to monitor if it's up or down. Scheduling is done through cron jobs.

```bash
# Install on unix/linux environment with
curl -s -S -L https://raw.githubusercontent.com/altlimit/statusalert/master/install.sh | bash

# Run against a valid *.http using VSCode REST Client
statusalert --http-file test.http
```

## Examples

You'll need smtpHost, smtpPort, smtpUser, smtpPass and alertEmails variables to be able to send alerts.

```
@smtpHost = smtp.gmail.com
@smtpPort = 587
@smtpUser = your@gmail.com
@smtpPass = app-password
# comma separated emails
@alertEmails = alert@gmail.com

@baseUrl = https://www.google.com

### status=200

GET {{baseUrl}}
```

This will check a GET request on your provided host and if status is not 200 it will send you an email once.
Once the site is back up then it will send you another message that it's up. It will only ever send you a message
whenever the state changes.

## Expected Results

You can use status, body and ignore for results.

```
### status=200&body=something
```
Means status code of 200 and body contains string 'something'.

You can ignore specific errors by adding it in ignore list comma separated. Errors are what you get in your alert email.

```
### ignore=timeout,bad
```

Means you ignore error message with timeout or bad.

## License

MIT