#!/usr/bin/env bash
source ./.env
cd ./sql/schema/
goose postgres $DB_URL up
