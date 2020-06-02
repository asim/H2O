# Log Package

This package is a drop-in replacement for seelog that allows you to add a source and context to your logging messages.

## Usage

Set log.Source at the start of your program. This will prefix all log messages with @source:

```
log.Source = "foo"
log.Infof("found it: %s", xyz)
```

outputs:

```
@foo found it: xyz
```

When you want to report a topic for every log message, create a new context and use that as your logger:

```
log.Source = "foo"
clog := log.NewContext("myid")
clog.Debugf("received request: %s", xyz)
```

outputs:

```
@foo #myid received request: xyz
```

You can also create sub-contexts:

```
sublog := clog.NewContext("process")
sublog.Debugf("doing something")
```

outputs:

```
@foo #myid #process doing something
```
