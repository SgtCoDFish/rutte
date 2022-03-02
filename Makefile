.PHONY: fresh
fresh:
	rm -rf content content-out
	cp -r /home/ashley/workspace/cert-manager-website/content ./content

.PHONY: test
test:
	go test -count=1 ./...

.PHONY: run
run:
	go run cmd/rutte/main.go

.PHONY: cleancheckout
cleancheckout: fresh run
	rm -rf $@
	git clone git@github.com:zentered/cert-manager.io.git $@
	cd $@ && git checkout feat/next-website
	rm -rf $@/content/en
	cp -r content-out/* $@/content/
	cd $@ && npm i


# uses https://github.com/raviqqe/muffet to check links
# to use, run make cleancheckout then in the cleancheckout dir run npm run dev
.PHONY: linkcheck
linkcheck:
	muffet http://localhost:3000/docs --exclude="gstatic.com|linkedin.com|googletagmanager.com|gstatic.com|github.com|github.io|googleapis.com|letsencrypt.org|amazon.com|gohugo.io|venafi.cloud|kubernetes.io|kyverno.io|k8s.io|cloudflare.com|cyberciti.biz|google.com|jetstack.io|eksctl.io|redhat.com|example.com|artifacthub.io|jetstack.net|slsa.dev|helm.sh"
