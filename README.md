Gow: A [Pow](http://pow.cx) fork for `Procfile` apps
================================================

Synopsis
--------
Gow is a zero-config development application server for Mac OS X. Have it serving your apps locally in under a minute.

      # do you trust me? if not, clone and read!
    $ curl https://raw.githubusercontent.com/jonasschneider/gow/master/dist/install.sh | sh
    $ cd ~/.pow
    $ ln -s /path/to/myapp

Introduction
------------

I love Pow. When developing systems of cascaded web applications, Pow keeps all the parts running for you, without having to mess with ports for each. It provides a lowest common denominator interface for starting Rack apps.

I also love [Foreman](https://github.com/ddollar/foreman). The `Procfile` is a great abstraction and concept that's useful to describe applications at a language-agnostic process level that fixes what seems to be an arbitrary restriction of Pow's -- that it can only run Ruby apps.

So, why not combine both? Gow works just like Pow, as a DNS server that resolves `*.dev` to its internal HTTP mux proxy. Running an application under Gow works exactly like Pow: simply symlink it to `~/.pow/<appname>`, and point your browser at `http://<appname>.dev`. However, while Pow looks for a `config.ru` file within the application's directory, Gow looks for a `Procfile` and starts a `web` process. All requests for the app are reverse-proxied to this process.

If you're on OS X, Gow provides Pow-like easy installation; run the provided `dist/install.sh` script to get started. On Linux, you might want to take a look at the install script for a snippet to steal to run Gow as a `daemontools` service, but you're going to have to fiddle with `/etc/resolv.conf` yourself.

Caveats
-------

WebSockets support is there, but not really tested or battle-hardened. The happy path *should* work, though.

Since launchd's logging is kinda shitty/dysfunctional, Gow writes its own log file to `~/Library/Logs/gowd.log`.

License
-------

Gow shares no runtime code with Pow. However, portions of the installation and set-up scripts have been modified for Gow. Gow therefore shares Pow's license (see `LICENSE`).

Contributing
------------
Patches welcome! The usual GitHub workflow applies.
