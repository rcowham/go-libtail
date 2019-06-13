# go-libtail
Tailing library based on fstab/grok_exporter tail package - cross compiles easily and doesn't lock Windows files!

Will evolve for use in other libraries such as p4prometheus and go-libp4dlog

## See cmd/tailer for a very simple command line interface showing very basic usage.

## Alternatives

Tried [papertrail/go-tail](https://github.com/papertrail/go-tail) but had two problems:
* reopen wasn't working properly
* log files was locked on Windows if being tailed

When I tried to import the files from [fstab/grok_exporter](https://github.com/fstab/grok_exporter) I found the Oniguruma 
dependency made life rather trickier - especially when cross compiling for Windows (even though the instructions all worked - just
took a long time!)

## ToDo

* Proper config
