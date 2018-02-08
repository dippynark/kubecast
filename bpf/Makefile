BPF = bpf_tty.o

#-I/usr/src/linux-headers-4.13.0-32/include 

CLANG ?= clang
CLANG_FLAGS := -D__KERNEL__ -D__ASM_SYSREG_H \
		-DCIRCLE_BUILD_URL=\"$(CIRCLE_BUILD_URL)\" \
		-Wno-unused-value \
		-Wno-pointer-sign \
		-Wno-compare-distinct-pointer-types \
		-Wunused \
		-Wall \
		-Werror \
		-O2 -emit-llvm \
		-Iinclude -I.

LLC ?= llc
LLC_FLAGS := -march=bpf -filetype=obj

LIB := $(shell find ./lib -name '*.h')

all: $(BPF)

%.o: %.c $(LIB)
	    ${CLANG} ${CLANG_FLAGS} -c $< -o - | ${LLC} ${LLC_FLAGS} -o $@; \
			#rm bpf_tty.o
