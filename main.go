package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"strconv"

	"github.com/labstack/echo"
)

type Student struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Email string `json:"email"`
}

var students = []Student{
	{ID: 1, Name: "Rahul", Age: 22, Email: "rahul@gmail.com"},
	{ID: 2, Name: "Alice", Age: 23, Email: "alice@gmail.com"},
	{ID: 3, Name: "Rob", Age: 24, Email: "rob@gmail.com"},
}

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	if err != nil {
		fmt.Printf("Invalid email: %s, error: %v\n", email, err)
	}
	return err == nil
}

func generateStudentSummary(student Student) (string, error) {
	prompt := fmt.Sprintf("Summarize the student's profile with the following information:\n\n"+
		"Name: %s\nAge: %d\nEmail: %s\n\nPlease keep the summary concise and in a friendly tone.",
		student.Name, student.Age, student.Email)

	payload := map[string]interface{}{
		"model":  "llama3.2",
		"prompt": prompt,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	resp, err := http.Post("http://localhost:11434/api/generate", "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var accumulatedResponse string

	decoder := json.NewDecoder(resp.Body)
	for {
		var part map[string]interface{}
		if err := decoder.Decode(&part); err != nil {
			return "", err
		}

		if response, ok := part["response"].(string); ok {
			accumulatedResponse += response
		}

		if done, ok := part["done"].(bool); ok && done {
			break
		}
	}

	if accumulatedResponse == "" {
		return "", fmt.Errorf("empty response from API")
	}

	fmt.Println("API Response: ", accumulatedResponse)
	return accumulatedResponse, nil
}

func main() {
	e := echo.New()

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Well, hello there!")
	})

	e.GET("/students", func(c echo.Context) error {
		return c.JSON(http.StatusOK, students)
	})

	e.GET("/students/:id", func(c echo.Context) error {
		sID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return err
		}

		for _, student := range students {
			if student.ID == sID {
				return c.JSON(http.StatusOK, student)
			}
		}
		return c.JSON(http.StatusNotFound, "Student not found")
	})

	e.POST("/students", func(c echo.Context) error {
		var reqBody Student
		if err := c.Bind(&reqBody); err != nil {
			return err
		}

		if reqBody.Name == "" {
			return c.JSON(http.StatusBadRequest, "Name is required")
		}
		if reqBody.Age <= 0 {
			return c.JSON(http.StatusBadRequest, "Age must be a positive integer")
		}
		if !isValidEmail(reqBody.Email) {
			return c.JSON(http.StatusBadRequest, "Invalid email format")
		}

		reqBody.ID = len(students) + 1
		students = append(students, reqBody)
		return c.JSON(http.StatusCreated, reqBody)
	})

	e.PUT("/students/:id", func(c echo.Context) error {
		sID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return err
		}

		var student *Student
		for i := range students {
			if students[i].ID == sID {
				student = &students[i]
				break
			}
		}

		if student == nil {
			return c.JSON(http.StatusNotFound, "Student not found")
		}

		var reqBody Student
		if err := c.Bind(&reqBody); err != nil {
			return err
		}

		if reqBody.Name == "" {
			return c.JSON(http.StatusBadRequest, "Name is required")
		}
		if reqBody.Age <= 0 {
			return c.JSON(http.StatusBadRequest, "Age must be a positive integer")
		}
		if !isValidEmail(reqBody.Email) {
			return c.JSON(http.StatusBadRequest, "Invalid email format")
		}

		student.Name = reqBody.Name
		student.Age = reqBody.Age
		student.Email = reqBody.Email
		return c.JSON(http.StatusOK, "student updated")
	})

	e.DELETE("/students/:id", func(c echo.Context) error {
		sID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return err
		}

		var index int
		var student *Student
		for i := range students {
			if students[i].ID == sID {
				student = &students[i]
				index = i
				break
			}
		}

		if student == nil {
			return c.JSON(http.StatusNotFound, "Student not found")
		}

		students = append(students[:index], students[index+1:]...)
		return c.JSON(http.StatusOK, "student deleted")
	})

	e.GET("/students/:id/summary", func(c echo.Context) error {
		sID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return err
		}

		var student *Student
		for _, s := range students {
			if s.ID == sID {
				student = &s
				break
			}
		}
		if student == nil {
			return c.JSON(http.StatusNotFound, "Student not found")
		}

		summary, err := generateStudentSummary(*student)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, "Failed to generate summary")
		}

		return c.JSON(http.StatusOK, map[string]string{"summary": summary})
	})

	e.Logger.Fatal(e.Start(":8080"))
}
