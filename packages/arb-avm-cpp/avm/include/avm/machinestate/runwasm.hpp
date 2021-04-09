/*
 * Copyright 2019-2020, Offchain Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#ifndef runwasm_hpp
#define runwasm_hpp

#include <wasm.h>

struct WasmEnvData {
    uint64_t buffer_len;
    Buffer buffer;
};

struct RunWasm {
    WasmEnvData data;
    wasm_func_t* run;
    wasm_trap_t* trap = NULL;
    RunWasm();

    std::pair<Buffer, uint64_t> run_wasm(Buffer buf, uint64_t len);

};

void run_wasm_test();

#endif /* runwasm_hpp */
