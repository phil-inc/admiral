# Adding a Command to Admiral

Admiral compiles to a binary. When you start Admiral, you
execute it by invoking the name of the binary:

```bash
./admiral
```

We add functionality to Admiral by adding additional commands.
For example, if you want to run the events or log controllers:

```bash
./admiral events
./admiral logs
```

So, the first step in making new functionality available is adding
a command.

## The cmd directory

The code for instanting commands at runtime lives in [cmd/][]. It
contains `main.go`, which defines all of the valid commands.
Individual commands are instantiated in `<command_name>.go`
(for example, `logs.go` & `events.go`).

## Adding a command

Start by creating the command file in [cmd/][]:

```bash
touch cmd/my_command.go
```


Add a function that will instantiate & return an object of type
`*cobra.Command`. This is from the libary `spf13/cobra`, a popular
CLI framework for Go. It should define some basic details about the
command & look something like this:

```go
func NewMyCommandCmd() *cobra.Command {
		return &cobra.Command{
			Use: "mycommand" # The string a user will pass to invoke the command
			Short: "Executes my command" # A short `help` description
			Long: `
		A longer description than my short description.
			`,
			RunE: MyCommandCmd, # A Golang function to execute on invocation
		}
}
```

Now, we need the function referenced in `RunE`. This function is what
actually gets executed by the application when the user invokes the
command. If you are writing a controller, this is where the controller
gets invoked.

```go
func MyCommandCmd(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			logrus.Warn("Too many arguments.")
			return fmt.Errorf("Too many arguments")
		}
		config := &config.Config{}

		if err := config.Load(configPath); err != nil {
			return err # If there's an error loading the config
		}

		if err := client.Run(config, "mycommand"); err != nil {
			return err # If there's an error executing the command
		}

		return nil
}
```

Lastly, we need to bind this function to the entrypoint of Admiral.
In `main.go`, add the function generating the command to the list of
functions generating commands for `rootCmd`:

```go
rootCmd.AddCommand(
	NewLogsCmd(),
	NewEventsCmd(),
	NewMyCommandCmd(),
```

Now, if we build & run `admiral`, information about the command will
appear in `help` and we can also invoke it by its `usage`.

[cmd/]: ./cmd
