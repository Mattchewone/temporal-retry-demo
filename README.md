### Steps to run this sample:
1) You need a Temporal service running. See details in README.md
2) Run the following command to start the worker
```
go run demo/worker/main.go
```
3) Run the following command to start the example
```
go run demo/starter/main.go
```

Edit the `res.json` from `{ "response": "error" }`, to `{ "response": "never" }` to trigger retries and then succeed. Leave as `error` if you wish for the retries to continue until the timeout happens