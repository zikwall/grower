#!/bin/bash

socat -u EXEC:'cat sample_test.log',pty,ctty UNIX-SENDTO:/tmp/syslog.sock