DB = postgresql://localhost/accrual?sslmode=disable

# How to create migration: `migrate create -ext sql -dir migrations -seq create_SOME_table`
migup:
	migrate -database ${DB} -path migrations up
migdrop:
	migrate -database ${DB} -path migrations drop
remig:
	make migdrop && make migup

lint:
	echo "goimports:" && goimports -l -local -w . && echo "gofumpt:" && gofumpt -l -w .

# tests
prep:
	cat m.json | http POST http://localhost:8888/api/goods ; \
	cat q.json | http POST http://localhost:8888/api/orders
run:
	go run cmd/gophermart/main.go -d=${DB}
runaccural:
	./cmd/accrual/accrual_darwin_amd64 \
	-a=":8888" \
	-d=${DB}
build:
	go build ./cmd/gophermart/...

# Update test template
upd:
	git fetch template && git checkout template/master .github
test:
	make build && \
	../go-autotests/bin/gophermarttest \
	-test.v -test.run=^TestGophermart/TestEndToEnd/ \
	-gophermart-binary-path=./gophermart \
	-gophermart-host=localhost \
	-gophermart-port=8080 \
	-gophermart-database-uri=${DB} \
	-accrual-binary-path=cmd/accrual/accrual_darwin_amd64 \
	-accrual-host=localhost \
	-accrual-port=$$(../go-autotests/bin/random unused-port) \
	-accrual-database-uri=${DB}
