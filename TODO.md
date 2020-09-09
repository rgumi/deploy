# TODO

1. finish switchover
2. add better methods for metric retrievel
3. test other strategies
4. add handlers for API
5. better workflow for creating routes/backends
6. dynamic reloading of routes when backend is added
   --- 7. add increasing counter for prom/dashboard

# Observed Errors

1. Storage Lock!?!?!?
   --- 2. Weird Weight update to [105, 251] when using switchover?!?!?! => uint8 overflow due to race condition in backend creation
