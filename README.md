Git Scan
========

A tool to scan git repositories for vulnerabilities.

Checks
------

- Suspicious file names, based on [Gitrob](https://github.com/michenriksen/gitrob)
- High entropy strings

Installation
------------

First initialize a PostgreSQL database with [database.sql](database/database.sql)

Install gitscan

    $ go get -v github.com/ardaxi/gitscan

Usage
-----

Grab a personal access token from Gitlab. See the [Gitlab documentation](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html) for more information.

If the database is not on the same host or needs a username/password, see [connection string parameters](https://godoc.org/github.com/lib/pq#hdr-Connection_String_Parameters) to create a DSN.

Then, start gitscan and let it run:

    $ gitscan -token $ACCESS_TOKEN -dsn "dbname=gitscan"

To see the results, start gitscan in server mode:

    $ gitscan -server -dsn "dbname=gitscan"

This will start a local server on port 8000 which you can reach over HTTP to provide your token and see results.
