pragma solidity ^0.4.0;

contract Pass1 {
    function add(int a, int b) public pure returns (int) {
        return a + b;
    }
}

/*
0      PUSH1  => 60
2      PUSH1  => 40
4      MSTORE
5      PUSH1  => 04
7      CALLDATASIZE
8      LT
9      PUSH1  => 3e
11     JUMPI
12     PUSH4  => ffffffff
17     PUSH29  => 0100000000000000000000000000000000000000000000000000000000
47     PUSH1  => 00
49     CALLDATALOAD
50     DIV
51     AND
52     PUSH4  => a5f3c23b
57     DUP2
58     EQ
59     PUSH1  => 43
61     JUMPI
62     JUMPDEST
63     PUSH1  => 00
65     DUP1
66     REVERT
67     JUMPDEST
68     CALLVALUE
69     ISZERO
70     PUSH1  => 4d
72     JUMPI
73     PUSH1  => 00
75     DUP1
76     REVERT
77     JUMPDEST
78     PUSH1  => 59
80     PUSH1  => 04
82     CALLDATALOAD
83     PUSH1  => 24
85     CALLDATALOAD
86     PUSH1  => 6b
88     JUMP
89     JUMPDEST
90     PUSH1  => 40
92     MLOAD
93     SWAP1
94     DUP2
95     MSTORE
96     PUSH1  => 20
98     ADD
99     PUSH1  => 40
101    MLOAD
102    DUP1
103    SWAP2
104    SUB
105    SWAP1
106    RETURN
107    JUMPDEST
108    ADD
109    SWAP1
110    JUMP
111    STOP
112    LOG1
113    PUSH6  => 627a7a723058
120    SHA3
121    Missing opcode 0xce
122    MULMOD
123    Missing opcode 0xf
124    Missing opcode 0xbb
125    COINBASE
126    RETURNDATASIZE
127    SMOD
128    LOG3
129    SWAP15
130    CALLDATACOPY
131    ADDRESS
132    BALANCE
133    SHL
134    DUP2
135    PUSH3  => a2d1cc
139    Missing opcode 0xfe
140    EQ
141    SWAP10
142    SMOD
143    RETURNDATACOPY
144    Missing opcode 0xcc
145    SSTORE
146    Missing opcode 0xed
147    COINBASE
148    CALLVALUE
149    Missing opcode 0xad
150    Missing opcode 0xf
151    ISZERO
152    BLOCKHASH
153    STOP
*/