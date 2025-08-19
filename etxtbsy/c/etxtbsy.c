#include <pthread.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <fcntl.h>
#include <stdlib.h>
#include <sys/file.h>
#include <sys/wait.h>
#include <errno.h>
#include <stdint.h>

void* runner(void*);

int
main(void)
{
	int i;
	pthread_t pid[20];

	for(i=1; i<20; i++)
		pthread_create(&pid[i], 0, runner, (void*)(uintptr_t)i);
	runner(0);
	return 0;
}

char script[] = "#!/bin/sh\nexit 0\n";

void*
runner(void *v)
{
	int i, fd, pid, status;
	char buf[100], *argv[2];

	i = (int)(uintptr_t)v;
	snprintf(buf, sizeof buf, "/var/tmp/fork-exec-%d", i);
	argv[0] = buf;
	argv[1] = 0;
	for(;;) {
		fd = open(buf, O_WRONLY|O_CREAT|O_TRUNC|O_CLOEXEC, 0777);
		if(fd < 0) {
			perror("open");
			exit(2);
		}
    write(fd, script, strlen(script));

    // -----------------------------------

    if (flock(fd, LOCK_EX) < 0) {
      perror("flock");
      exit(2);
    }
    close(fd);
    fd = open(buf, O_RDONLY|O_CLOEXEC, 0777);
    if(fd < 0) {
      perror("open (readonly)");
      exit(2);
    }
    if (flock(fd, LOCK_SH) < 0) {
      perror("flock (readonly)");
      exit(2);
    }

    // -----------------------------------

    close(fd);
    pid = fork();
		if(pid < 0) {
			perror("fork");
			exit(2);
		}
		if(pid == 0) {
			execve(buf, argv, 0);
			exit(errno);
		}
		if(waitpid(pid, &status, 0) < 0) {
			perror("waitpid");
			exit(2);
		}
		if(!WIFEXITED(status)) {
			perror("waitpid not exited");
			exit(2);
		}
		status = WEXITSTATUS(status);
		if(status != 0)
			fprintf(stderr, "exec: %d %s\n", status, strerror(status));
	}
	return 0;
}
