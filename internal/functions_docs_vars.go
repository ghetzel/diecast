package internal

func loadRuntimeFunctionsVariables(server ServerProxy) FuncGroup {
	return FuncGroup{
		Name: `Dynamic Variables`,
		Description: `A set of functions that allow for custom data to be set, retrieved, and removed at runtime; ` +
			`providing greater flexibility over standard template variables. All variables created or modified using ` +
			`these functions are accessible under the global _$.vars_ object.  For example, a variable set with ` +
			`<code>{{ var "test" 123 }}</code> would be retrieved with <code>{{ $.vars.test }}</code>, which would ` +
			`contain the integer value _123_.`,
		Functions: []FuncDef{
			{
				Name: `var`,
				Summary: `Declare a new variable with a given name, optionally setting it to an initial value.  ` +
					`If a value is not provided, the variable is set to a null (empty) value. You can use this ` +
					`behavior to clear out the value of an existing variable. The string defining the variable ` +
					`name is interpreted as a "dot.separated.path" that can be used to set the value in a deeply-nested objects.`,
				Arguments: []FuncArg{
					{
						Name:        `name`,
						Type:        `string`,
						Description: `The name of the variable to declare or set.`,
					}, {
						Name:        `value`,
						Type:        `any`,
						Optional:    true,
						Description: `If specified, the value at _$.vars.NAME_ will be set to the given value.  Otherwise, it will be set to _null_.`,
					},
				},
				Examples: []FuncExample{
					{
						Code:   `var "test"`,
						Return: nil,
					}, {
						Code:   `var "test" "Hello"`,
						Return: `Hello`,
					}, {
						Code: `var "this.is.a.value" true`,
						Return: map[string]interface{}{
							`this`: map[string]interface{}{
								`is`: map[string]interface{}{
									`a`: map[string]interface{}{
										`value`: true,
									},
								},
							},
						},
					},
				},
			}, {
				Name: `push`,
				Summary: `Append a value to an array residing at the named variable.  If the current value is nil, or ` +
					`the variable does not exist, the variable will be created as an array containing the provided value.` +
					`If the current value exists but is not already an array, it will first be converted to one, to which the ` +
					`given value will be appended.`,
				Arguments: []FuncArg{
					{
						Name:        `name`,
						Type:        `string`,
						Description: `The name of the variable to append to.`,
					}, {
						Name:        `value`,
						Type:        `any`,
						Description: `The value to append.`,
					},
				},
				Examples: []FuncExample{
					{
						Code:   `push "test" 123`,
						Return: []int{123},
					}, {
						Code:   `push "test" 456`,
						Return: []int{123, 456},
					}, {
						Code: `push "users.names" "Bob"`,
						Return: map[string]interface{}{
							`users`: map[string]interface{}{
								`names`: []string{`Alice`, `Bob`},
							},
						},
					},
				},
			}, {
				Name: `pop`,
			}, {
				Name: `varset`,
			}, {
				Name: `increment`,
			}, {
				Name: `incrementByValue`,
			},
		},
	}
}
