persistent-logging-plugin
*************************

``persistent-logging-plugin`` is an extension of the ``local`` file logging driver that captures output not per container
but per image name. This ensures that the logs persists, even if you frequently release and prune old containers.

Installing and enabling persistent-logging-plugin
=================================================

1. Clone the repository::

    $ git clone git@github.com:AlbinLindskog/docker_persistent-logging-plugin.git

2. The plugin is built inside a docker image and is then exported. There is an makefile to facilitate the process.
   Simply run::

    $ make all

3. Enable the plugin::

    $ docker plugin enable persistent-logging-plugin:0.0.1

4. Restart the docker daemon for the plugin to be available::

    $ service docker restart

5. Ensure that it has been installed correctly::

    $ docker plugin ls

    ID             NAME                                DESCRIPTION                                      ENABLED
    e2c4e98644d3   persistent-logging-plugin:0.0.1     docker logging plugin that persists the logs…    false

Configuring persistent-logging-plugin
=====================================

Option 1
--------
The daemon.json file allows you to configure all containers with the same options. For example::

    {
        "log-driver": "persistent-logging-plugin:0.0.1",
        "log-opts": {
            "max-size": "10m"
        }
    }


Option 2:
--------
Configure the plugin separately for each container when using the docker run command. For example::

    $ docker run --log-driver persistent-logging-plugin:0.0.1 --log-opt max-size=10m ...

Option 3:
--------
 Configure the plugin separately for each container when using docker-compose. For example::

    my-container:
      ...
      log_driver: "persistent-logging-plugin:0.0.1"
      log_opt:
         max-size: "10m"

Supported options:
------------------
.. list-table::
   :widths: 10 70 20
   :header-rows: 1

   * - Option
     - Description
     - Example
   * - max-size
     - The maximum size of the log before it is rolled. A positive integer plus a modifier representing the unit of measure (k, m, or g). Defaults to 20m.
     - ``--log-opt max-size=10m``
   * - max-file
     - The maximum number of log files that can be present. If rolling the logs creates excess files, the oldest file is removed. A positive integer. Defaults to 5.
     - ``--log-opt max-file=3``
   * - compress
     - Toggle compression of rotated log files. Enabled by default.
     - ``--log-opt compress=false``

Caveats:
--------
Docker plugins are ran as separate processes by the docker daemon, with separate filesystem. As such is it not possible
to specify the location on the host filesystem where the logs will be stored. They are stored at
``/var/lib/docker/plugins/<plugin id>/rootfs/var/logs``
It might be possible to setup a bind mount for this location, however that will need to be specified in config.json,
and would not be a very flexible option.

The logs stored by the ``local`` file logging driver, and by extension``persistent-logging-plugin``, are intended to
only be accessed through the driver that created them via the ``docker logs`` command, so it is not a feature worth
spending time on, imo.

