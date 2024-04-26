clean:
	-rm -r build

test:
	go test ./cmdlang/...

site: clean
	mkdir build
	mkdir build/site
	cp -r _site/* build/site/.
	GOOS=js GOARCH=wasm go build -o build/site/playwasm.wasm ./cmd/playwasm/.

site-deploy: site
	netlify deploy --dir build/site --prod