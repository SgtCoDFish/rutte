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
	git clone git@github.com:cert-manager/website.git $@
	cd $@ && git checkout next-website-base
	rm -rf $@/content/en
	cp -r content-out/* $@/content/
	cd $@ && npm i
	echo "{\"presets\": [\"next/babel\"]}" > $@/.babelrc

#LINKURL=https://deploy-preview-896--cert-manager-website.netlify.app/docs
LINKURL=http://localhost:3000/v1.8-docs

# uses https://github.com/raviqqe/muffet to check links
# to use, run make cleancheckout then in the cleancheckout dir run npm run dev
.PHONY: linkcheck
linkcheck:
	muffet $(LINKURL) --timeout=30 --rate-limit=10 --exclude="gstatic.com|linkedin.com|googletagmanager.com|gstatic.com|github.com|github.io|googleapis.com|letsencrypt.org|amazon.com|gohugo.io|venafi.com|vaultproject.io|venafi.cloud|kubernetes.io|kyverno.io|k8s.io|cloudflare.com|cyberciti.biz|google.com|jetstack.io|eksctl.io|redhat.com|example.com|artifacthub.io|jetstack.net|slsa.dev|helm.sh|ietf.org|cert-manager.io|operatorframework.io|openshift.com|tldrlegal.com|twitter.com"
