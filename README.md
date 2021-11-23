## Status Alert

A simple http request checker to monitor if it's up or down. Scheduling is done through cron jobs.

```bash
# Run against a valid *.http using VSCode REST Client
statusalert --http-file test.http
```

You'll need smtpHost, smtpPort, smtpUser, smtpPass and alertEmails variables to be able to send alerts.