package main

func main() {
	rand.Seed(time.Now().UTC().UnixNano()) //Seed the RNG to get different values
	fmt.Println("Random number between 1 and 100 ->", randomInteger(1, 100))
}

func randomInteger(min int, max int) int {
	return min + rand.Intn(max-min)
}
