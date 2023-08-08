deploy: SHELL := /bin/zsh


deploy:
	kurtosis run --enclave ethTestnet ./eth2-package '$(shell cat ./config.json)'

clean:
	kurtosis clean -a