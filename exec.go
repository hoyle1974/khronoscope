package main

// // Create an exec request
// req := clientset.CoreV1().RESTClient().
// 	Post().
// 	Resource("pods").
// 	Namespace(namespace).
// 	Name(podName).
// 	SubResource("exec").
// 	Param("container", containerName).
// 	Param("stdout", "true").
// 	Param("stdin", "true").
// 	Param("stderr", "true").
// 	Param("tty", "true")
// for _, cmd := range command {
// 	req.Param("command", cmd)
// }

// // Create an executor
// exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
// if err != nil {
// 	log.Fatalf("Failed to initialize executor: %v", err)
// }

// // Set up input and output streams
// stdin := os.Stdin
// stdout := os.Stdout
// stderr := os.Stderr

// // Execute command in the container
// err = exec.Stream(remotecommand.StreamOptions{
// 	Stdin:  stdin,
// 	Stdout: stdout,
// 	Stderr: stderr,
// 	Tty:    true,
// })
// if err != nil {
// 	log.Fatalf("Failed to execute command: %v", err)
// }

// fmt.Println("Command executed successfully")
