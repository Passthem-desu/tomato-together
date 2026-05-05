fix_format:
	cd backend && gofmt -w .
	cd frontend && npm run format
