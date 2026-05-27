.PHONY: up down logs restart compile-contract copy-bytecode tidy

up:
	docker compose up --build

down:
	docker compose down

logs:
	docker compose logs -f

restart:
	docker compose down
	docker compose up --build

compile-contract:
	node scripts/compile-contract.js

copy-bytecode:
	cp contracts/build/Voting.bin backend/contract/Voting.bin

tidy:
	cd backend && go mod tidy
