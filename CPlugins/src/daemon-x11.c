#include <time.h>
#include <signal.h>
#include <unistd.h>
#include <X11/Xlib.h>
#include <X11/extensions/XInput2.h>
#include <stdio.h>
#include <dirent.h>
#include <string.h>
#include <stdlib.h>
#include <limits.h>

int find_pid_by_name(const char *name) {
    DIR *dir;
    struct dirent *entry;
    FILE *fp;
    char path[PATH_MAX];
    char cmdline[PATH_MAX];
    char *base_name;
    int pid = -1;

    dir = opendir("/proc");
    if (!dir) {
        perror("Failed to open /proc");
        return -1;
    }

    while ((entry = readdir(dir)) != NULL) {
        // Skip non-PID directories
        if (entry->d_type != DT_DIR)
            continue;

        char *endptr;
        long pid_candidate = strtol(entry->d_name, &endptr, 10);
        if (*endptr != '\0')  // Not a valid PID
            continue;

        // Build path to cmdline file
        snprintf(path, sizeof(path), "/proc/%ld/cmdline", pid_candidate);

        fp = fopen(path, "r");
        if (!fp)
            continue;

        if (fgets(cmdline, sizeof(cmdline), fp) != NULL) {
            // cmdline contains the complete command with arguments separated by null bytes
            // We'll just look for the basename of the executable
            base_name = strrchr(cmdline, '/');
            base_name = base_name ? base_name + 1 : cmdline;

            // Compare with our target name
            if (strcmp(base_name, name) == 0) {
                pid = (int)pid_candidate;
                fclose(fp);
                break;
            }
        }
        fclose(fp);
    }

    closedir(dir);
    return pid;
}

static volatile sig_atomic_t shutdown_flag = 0;

void mouse_shutdown() {
   shutdown_flag = 1;
}

int monitorMouse(int pid) {
    Display *dpy = XOpenDisplay(NULL);
    if (!dpy) {
        return 1;
    }
    int xi_opcode, event, error;
    if (!XQueryExtension(dpy, "XInputExtension", &xi_opcode, &event, &error)) {
        return 1;
    }
    int major = 2, minor = 0;
    if (XIQueryVersion(dpy, &major, &minor) != Success) {
        return 1;
    }
    Window root = DefaultRootWindow(dpy);
    XIEventMask mask;
    unsigned char mask_bits[XIMaskLen(XI_LASTEVENT)] = {0};
    mask.deviceid = XIAllDevices; 
    mask.mask_len = sizeof(mask_bits);
    mask.mask = mask_bits;
    XISetMask(mask_bits, XI_RawButtonPress);
    XISelectEvents(dpy, root, &mask, 1);
    XFlush(dpy);
    XEvent ev;
    struct timespec last_event_time = {0, 0};
    const long debounce_time_ns = 250000000L; // 0.25 seconds in nanoseconds
    while (shutdown_flag == 0) {
        XNextEvent(dpy, &ev);

        if (ev.xcookie.type != GenericEvent || 
            ev.xcookie.extension != xi_opcode) {
            continue;
        }

        if (XGetEventData(dpy, &ev.xcookie)) {
            if (ev.xcookie.evtype == XI_RawButtonPress) {
               struct timespec current_time;
               clock_gettime(CLOCK_MONOTONIC, &current_time);
               // Calculate time difference in nanoseconds
               long time_diff = (current_time.tv_sec - last_event_time.tv_sec) * 1000000000L;
               time_diff += current_time.tv_nsec - last_event_time.tv_nsec;

               if (time_diff >= debounce_time_ns || last_event_time.tv_sec == 0) {
                   kill(pid, SIGUSR1);
                   last_event_time = current_time;
               }
            }
            XFreeEventData(dpy, &ev.xcookie);
        }
    }

    XCloseDisplay(dpy);
    return 0;
}

int main() {
    const char *process_name = "wm";
    int pid = find_pid_by_name(process_name);

    if (pid != -1) {
        printf("Found '%s' with PID: %d\n", process_name, pid);
        monitorMouse(pid);
    } else {
        printf("Process '%s' not found\n", process_name);
    }

    return 0;
}
