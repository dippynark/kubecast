#include <linux/types.h>

#define BUFSIZE 256

struct tty_write_t {
    __u64 timestamp;
    __u32 count;
    char buf[BUFSIZE];
};

struct tty_t {
    unsigned long ino;
}