PLUGIN_NAME=persistent-logging-plugin
PLUGIN_TAG=0.0.1

all: clean docker rootfs create

clean:
	rm -rf ./plugin/rootfs

docker:
	docker build -t ${PLUGIN_NAME}:rootfs .

rootfs:
	mkdir -p ./plugin/rootfs
	docker create --name tmprootfs ${PLUGIN_NAME}:rootfs
	docker export tmprootfs | tar -x -C ./plugin/rootfs
	docker rm -vf tmprootfs

create:
	docker plugin rm -f ${PLUGIN_NAME}:${PLUGIN_TAG} || true
	docker plugin create ${PLUGIN_NAME}:${PLUGIN_TAG} ./plugin
