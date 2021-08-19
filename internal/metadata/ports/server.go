package ports

func StartServer() {
	r := NewMuxRouter()
	r.POST("data", addMetadata)
	r.Serve(":9090")
}
