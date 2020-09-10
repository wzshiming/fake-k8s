

.PHONY: install
install:
	docker-compose up -d
	sleep 2
	kubectl apply -f node.yaml
	sleep 2
	kubectl get node

.PHONY: demo
demo:
	kubectl apply -f demo
	sleep 1
	kubectl get pod

.PHONY: uninstall
uninstall:
	docker-compose down
	rm -rf etcd-data


