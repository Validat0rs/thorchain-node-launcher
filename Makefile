add-node-launcher:
	git subtree add --prefix=node-launcher https://gitlab.com/thorchain/devops/node-launcher.git master --squash
	@git add --all

update-node-launcher:
	git subtree pull https://gitlab.com/thorchain/devops/node-launcher.git master --prefix=node-launcher --squash
