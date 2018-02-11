#include <linux/types.h>

#define BUFSIZE 256

struct tty_write_t {
    __u32 count;
    char buf[BUFSIZE];
    __u32 sessionid;
    __u64 timestamp;
};

struct sid_t {
    int sid; 
};
