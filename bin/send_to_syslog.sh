#!/bin/bash

for i in `seq 1 100`;
do
    socat -u EXEC:'cat sample_test.log',pty,ctty UNIX-SENDTO:/tmp/syslog.sock
done