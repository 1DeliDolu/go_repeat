package view

// SSR form re-render için (password'u geri basmayacağız)
type LoginForm struct {
	Email string
}

// SignupForm holds form data for signup page re-render
type SignupForm struct {
	Email string
}
