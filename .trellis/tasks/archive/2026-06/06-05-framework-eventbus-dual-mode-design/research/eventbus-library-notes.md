# EventBus Library Notes

## Library

Package:

```text
github.com/asaskevich/EventBus
```

Reference:

* `https://github.com/asaskevich/EventBus`
* `https://pkg.go.dev/github.com/asaskevich/EventBus`

## Useful Capabilities

EventBus provides in-process publish/subscribe primitives:

* create a bus with `EventBus.New()`;
* subscribe callbacks to a topic with `Subscribe`;
* publish a topic with `Publish`;
* subscribe async callbacks with `SubscribeAsync`;
* wait for async callbacks with `WaitAsync`;
* unsubscribe callbacks when needed.

This fits the module decoupling requirement because modules can register
callbacks by topic instead of being called directly by the usecase.

## Constraints Relevant to This Project

EventBus is not a transaction manager:

* it does not know about `fwusecase.WithAppTx`;
* it does not persist events;
* it does not persist handler execution state;
* it does not provide retry/dead-letter/replay;
* raw `Publish` does not give the framework a typed per-subscriber error model.

Therefore the project must wrap EventBus in `api/framework/events`.

## Design Implications

Use EventBus as the dispatch engine, not as the public API.

Framework responsibilities:

* hide the external dependency from routes/usecases/models;
* define typed `Event`, `Subscription`, `AsyncHandler`, and `TxHandler`
  contracts;
* route subscribers by delivery mode;
* run sync handlers through EventBus synchronous callbacks;
* run async handlers through EventBus async callbacks;
* collect sync handler errors through a framework-owned dispatch object;
* log async handler failures;
* delay async events until after app transaction commit when needed.

## Recommended Usage Shape

Sync transaction handlers:

```go
bus.Subscribe(syncTopic(event.Topic), func(dispatch *SyncDispatch) {
    err := handler.Handle(dispatch.TxContext, dispatch.Event)
    dispatch.Record(subscription, err)
})
```

Async best-effort handlers:

```go
bus.SubscribeAsync(asyncTopic(event.Topic), func(dispatch *AsyncDispatch) {
    if err := handler.Handle(dispatch.Context, dispatch.Event); err != nil {
        dispatch.Log(subscription, err)
    }
}, false)
```

The exact `SubscribeAsync` options should be verified during implementation
against the library version added to `go.mod`.
