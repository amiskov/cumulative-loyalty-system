DB = postgresql://localhost/accrual?sslmode=disable

# How to create migration: `migrate create -ext sql -dir migrations -seq create_SOME_table`
migup:
	migrate -database ${DB} -path migrations up
migdrop:
	migrate -database $$(DB) -path migrations drop
remig:
	make migdrop && make migup

run:
	go run cmd/gophermart/main.go
runaccural:
	./cmd/accrual/accrual_darwin_amd64 \
	-a=":8888" \
	-d="postgresql://localhost/accrual?sslmode=disable"
build:
	go build ./cmd/gophermart/...

# Update test template
upd:
	git fetch template && git checkout template/master .github
test:
	make build && \
	../go-autotests/bin/gophermarttest \
	-test.v -test.run=^TestGophermart/TestEndToEnd/login_user \
	-gophermart-binary-path=./gophermart \
	-gophermart-host=localhost \
	-gophermart-port=8080 \
	-gophermart-database-uri="postgresql://localhost/gophermart_cls?sslmode=disable" \
	-accrual-binary-path=cmd/accrual/accrual_darwin_amd64 \
	-accrual-host=localhost \
	-accrual-port=$$(../go-autotests/bin/random unused-port) \
	-accrual-database-uri="postgresql://localhost/accrual?sslmode=disable"
