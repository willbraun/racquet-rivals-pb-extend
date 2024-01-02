# Overview

This repo is for the backend of [racquetrivals.com](https://www.racquetrivals.com). The app uses Pocketbase, a BaaS that provides a SQLite database and built-in APIs to interact with it.

Pocketbase offers a way to extend its functionality by importing it into a new Go package, and adding custom logic to that new package ([documentation](https://pocketbase.io/docs/go-overview/)). This repo includes logic to award points to users who have correctly predicted a match result. 

For more information on Racquet Rivals, see this [repo](https://github.com/willbraun/tennis-bracket-frontend).

# Technologies Used

Go, Pocketbase
