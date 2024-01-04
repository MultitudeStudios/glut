docker exec -i postgres psql postgres://postgres:postgres@localhost:5432 < db/init.sql

# auth database
docker exec -i postgres psql postgres://postgres:postgres@localhost:5432/glut < db/auth.sql
docker exec -i postgres psql postgres://postgres:postgres@localhost:5432/glut < db/auth.test.sql
