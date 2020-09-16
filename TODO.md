# TODO

- [x] == solved
- [ ] == not solved

## Tasks

- [x] finish switchover
- [x] add better methods for metric retrievel
- [x] test other strategies
- [x] add handlers for API
- [x] better workflow for creating routes/backends
- [x] dynamic reloading of routes when backend is added
- [x] add increasing counter for prom/dashboard

## Observed Errors

- [x] Storage Lock
- [x] Weird Weight update to [105, 251] when using switchover?! => uint8 overflow due to race condition in backend creation
- [x] initial healthchecks are sometimes not executed when multiple routes are added => race condition in metrics-job
