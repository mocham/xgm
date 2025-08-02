docker run -it -v./:/src --workdir /src --rm --entrypoint /usr/bin/gcc make -o mouse_monitor daemon-x11.c -lXi -lX11
