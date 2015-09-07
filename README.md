Gow: A development server for `Procfile` apps
=============================================

Synopsis
--------
Gow is a zero-config development application server for Mac OS X. Have it serving your apps locally in under a minute.

      # do you trust me? if not, clone and read!
    $ curl https://raw.githubusercontent.com/jonasschneider/gow/master/dist/install.sh | sh
    $ cd ~/.pow
    $ ln -s /path/to/myapp myapp
    $ open http://myapp.dev

Introduction
------------

[Pow](http://pow.cx) is a simple tool to run development environments for systems composed of multiple Rack-based web applications and services. Transparently and zero-config, it sets up a wildcard `<appname>.dev` domain and manages the server processes for each application.

Gow generalises this approach beyond running Ruby apps. Gow runs any application with a `Procfile` (as pioneered by [Heroku](https://heroku.com)), no matter what language it's in. Simple. Unlike Pow, it also supports Websocket connections.

Internally, Gow works just like Pow, as a DNS server that resolves `*.dev` to its internal HTTP multiplexing proxy. Running an application under Gow works exactly the same: simply symlink it to `~/.pow/<appname>`, and point your browser at `http://<appname>.dev`. However, while Pow looks for a `config.ru` file within the application's directory, Gow looks for a `Procfile` and starts a `web` process. All requests for the app are reverse-proxied to this process.

If you're on OS X, Gow provides Pow-like easy installation; run the provided `dist/install.sh` script to get started. On Linux, you might want to take a look at the install script for a snippet to run Gow under the `init` of your choice, and you'll have to mess with `/etc/resolv.conf` yourself.

Caveats
-------

Since launchd's logging is kinda shitty/dysfunctional, Gow writes its own log file to `~/Library/Logs/gowd.log`.

License
-------

Gow shares no runtime code with Pow. However, portions of the installation and set-up scripts have been modified for Gow. Gow therefore shares Pow's license (see `LICENSE`).

Contributing
------------
Patches welcome! The usual GitHub workflow applies.
