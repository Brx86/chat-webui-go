package main

func main() {
	initModels()
	r := setupRouter()
	r.Run(":8888")
}
