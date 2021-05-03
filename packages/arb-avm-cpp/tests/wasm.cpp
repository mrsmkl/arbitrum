
#include <catch2/catch.hpp>
#include <avm_values/vmValueParser.hpp>
#include <avm_values/code.hpp>
#include <avm_values/value.hpp>
#include <avm/machinestate/machinestate.hpp>
#include <fstream>
#include <iostream>

#include <data_storage/arbstorage.hpp>
#include <avm/machinestate/runwasm.hpp>
#include <boost/algorithm/hex.hpp>

/*
value get_immed_value(uint8_t *a) {
    return 0;
}*/

value get_int_value(std::vector<uint8_t> bytes, uint64_t offset) {
    auto acc = 0;
    for (int i = 0; i < 8; i++) {
        acc = acc*256;
        acc += bytes[offset+i];
    }
    return acc;
}

TEST_CASE("wasm_compile") {
    SECTION("Code to hash") {
        std::string hexstr = "3b00003300003900003000003900005001000000000000000300320000500100000000000000020032000050010000000000000004003200003b00003b00013b00017300003b00013400003000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c010000000000000003004300001601000000000000000700400000390000500100000000000000010032000038000053010000000000000001003300005101000000000000000200320000a40000a2010000000000000000005301000000000000000000a201000000000000000800530100000000000000000050010000000000000002003200003b00003b00013400003000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c01000000000000000300430000160100000000000000070040000039000050010000000000000001003200003800005301000000000000000100a10000a201000000000000000000530100000000000000000050010000000000000002003200003b00003b00013400003000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c0100000000000000030043000016010000000000000007004000003900005001000000000000000100320000380000530100000000000000010033000051010000000000000003003b0000320000a20100000000000000000053010000000000000000003b00013400003000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c0100000000000000030043000016010000000000000007004000003900005001000000000000000100320000380000530100000000000000010050010000000000000003003b00003200003b00013400003000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c010000000000000003004300001601000000000000000700400000390000500100000000000000010032000038000053010000000000000001003b00013b00013b00003b010000000000000000003b00003000003900003b00013400003000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c0100000000000000030043000016010000000000000007004000003b0100000000000000070050010000000000000001003200005401000000000000000000a50100000000000000000043000053010000000000000000005401000000000000000000a501000000000000000800430000530100000000000000000038030000000000000001003b00003b0100000000000000de003b00003b010000000000000000003b00003b00013400003000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c0100000000000000030043000016010000000000000007004000004300005000004300004400001c0100000000000000030043000016010000000000000007004000003b0100000000000000000050010000000000000001003200005401000000000000000000a5010000000000000000003b0100000000000000020053010000000000000000003803000000000000000a003300005101000000000000000000320000a5010000000000000000003b01000000000000040000a000003300005101000000000000000300510100000000000000020051010000000000000001003b02003800003200003b00003b00003b0000ff";
        std::vector<uint8_t> bytes;
        bytes.resize(hexstr.size() / 2);
        boost::algorithm::unhex(hexstr.begin(), hexstr.end(), bytes.begin());
        // code to hash
        auto code = std::make_shared<Code>(0);
        CodePointStub stub = code->addSegment();
        std::vector<value> labels;
        int i = 0;
        while (bytes[i] != 255) {
            OpCode opcode = static_cast<OpCode>(bytes[i]);
            i++;
            Operation op = {opcode};
            auto immed = bytes[i];
            i++;
            if (immed == 1) {
                op = {opcode, get_int_value(bytes, i)};
                i += 8;
            } else if (immed == 2) {
                std::vector<value> v;
                v.push_back(Buffer());
                v.push_back(0);
                v.push_back(Buffer());
                v.push_back(0);
                v.push_back(100000); // check that these are the same
                op = {opcode,Tuple::createTuple(v)};
            } else if (immed == 3) {
                std::vector<value> v;
                v.push_back(Buffer());
                v.push_back(get_int_value(bytes, i));
                i += 8;
                op = {opcode,Tuple::createTuple(v)};
            }
            stub = code->addOperation(stub.pc, op);
            if (op.immediate) {
                std::cerr << "Immed hash " << op << " hash " << intx::to_string(hash_value(*op.immediate), 16) << "\n";
            }
            std::cerr << "Loaded op " << op << " hash " << intx::to_string(stub.hash, 16) << "\n";
            if (bytes[i]) {
                // std::cerr << "Label " << stub << " at " << labels.size() <<
                // "\n";
                labels.push_back(stub);
            }
            i++;
        }

        std::reverse(labels.begin(), labels.end());
        auto table = make_table(labels);
        std::cerr << "Here " << intx::to_string(stub.hash, 16) << " "
                  << labels.size() << " \n";
        // std::cerr << "Table " << table << " hash " <<
        // intx::to_string(hash_value(table), 16) << "\n";
        std::cerr << "Table hash " << intx::to_string(hash_value(table), 16)
                  << " size " << getSize(table) << "\n";
        // convert table
        std::cerr << "Buffer hash " << intx::to_string(hash_value(Buffer()), 16) << "\n";
    }
}

TEST_CASE("Wasm") {
    SECTION("Code to hash") {
        /*

        auto res = run_wasm(Buffer(), 123);

        auto storage = ArbStorage("/home/sami/tmpstorage");
        // auto state = makeWasmMachine(123, Buffer());
        storage.initialize("/home/sami/arb-os/wasm-inst.json");

        auto arbcore = storage.getArbCore();
        arbcore->startThread();

        ValueCache value_cache{1, 0};
        auto cursor = arbcore->getExecutionCursor(10000000, value_cache);
        std::cerr << "Status: " << cursor.status.code() << "\n";
        std::cerr << "gas used: " << cursor.data->getOutput().arb_gas_used << "\n";
        std::cerr << "steps: " << cursor.data->getOutput().total_steps << "\n";
*/

        auto storage = ArbStorage("/home/sami/tmpstorage");
        auto state = makeWasmMachine(123, Buffer());
        storage.initialize(state);


        std::cerr << "Starting " << intx::to_string(state.hash().value(), 16) << "\n";

        uint256_t gasUsed = runWasmMachine(state);

        std::cerr << "Stopping " << intx::to_string(state.hash().value(), 16) << " gas used " << gasUsed << "\n";

        OneStepProof proof;
        state.marshalWasmProof(proof);
        std::cerr << "Made proof " << proof.buffer_proof.size() << "\n";
        marshal_uint256_t(gasUsed, proof.buffer_proof);
    }

}
