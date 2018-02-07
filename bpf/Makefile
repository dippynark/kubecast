BPF = bbf_tty.o

#-I/usr/src/linux-headers-4.13.0-32/include 
FLAGS :=  -Iinclude -I. -D__NR_CPUS__=$(shell nproc) -O2

CLANG ?= clang
CLANG_FLAGS :=  ${FLAGS} -emit-llvm
# eBPF verifier enforces unaligned access checks where necessary, so don't
# # let clang complain too early.
CLANG_FLAGS += -Wall

LLC   ?= llc
LLC_FLAGS   := -march=bpf -filetype=obj

LIB := $(shell find ./lib -name '*.h')

all: $(BPF)

%.o: %.c $(LIB)
	    ${CLANG} ${CLANG_FLAGS} -c $< -o - | ${LLC} ${LLC_FLAGS} -o $@; \
			