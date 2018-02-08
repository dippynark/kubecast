BPF = bpf_tty.o

#-I/usr/src/linux-headers-4.13.0-32/include 

CLANG ?= clang
CLANG_FLAGS := -O2 \
	-target bpf \
	-emit-llvm \
	-Wno-address-of-packed-member \
	-Wno-unknown-warning-option \
	-Iinclude -I. \
	-D__NR_CPUS__=$(shell nproc) \
	-DENABLE_ARP_RESPONDER \
	-DHANDLE_NS
# eBPF verifier enforces unaligned access checks where necessary, so don't
# # let clang complain too early.
#CLANG_FLAGS += -Wall

LLC ?= llc
LLC_FLAGS := -march=bpf -filetype=obj #-mcpu=probe 

LIB := $(shell find ./lib -name '*.h')

all: $(BPF)

%.o: %.c $(LIB)
	    ${CLANG} ${CLANG_FLAGS} -c $< -o - | ${LLC} ${LLC_FLAGS} -o $@; \
			#rm bpf_tty.o
