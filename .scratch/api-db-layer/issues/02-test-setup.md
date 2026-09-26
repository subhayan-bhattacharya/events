# Test setup

Type: grilling
Status: open
Blocked by:

## Question

How do tests get a database? Options: testcontainers-go (fresh Postgres per test run)
versus the docker-compose Postgres. Also: how is each test isolated (transaction
rollback, truncate, or schema per test), and how are SQL-level constraint checks run?
