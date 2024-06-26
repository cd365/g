package g

// Err panic if err != nil
func Err(err error) {
	if err != nil {
		panic(err)
	}
}
