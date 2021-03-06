/* Generated by ts-generator ver. 0.0.8 */
/* tslint:disable */

import { Contract, ContractTransaction, EventFilter, Signer } from 'ethers'
import { Listener, Provider } from 'ethers/providers'
import { Arrayish, BigNumber, BigNumberish, Interface } from 'ethers/utils'
import {
  TransactionOverrides,
  TypedEventDescription,
  TypedFunctionDescription,
} from '.'

interface ArbERC20Interface extends Interface {
  functions: {
    approve: TypedFunctionDescription<{
      encode([spender, amount]: [string, BigNumberish]): string
    }>

    totalSupply: TypedFunctionDescription<{ encode([]: []): string }>

    transferFrom: TypedFunctionDescription<{
      encode([sender, recipient, amount]: [
        string,
        string,
        BigNumberish
      ]): string
    }>

    increaseAllowance: TypedFunctionDescription<{
      encode([spender, addedValue]: [string, BigNumberish]): string
    }>

    balanceOf: TypedFunctionDescription<{
      encode([account]: [string]): string
    }>

    decreaseAllowance: TypedFunctionDescription<{
      encode([spender, subtractedValue]: [string, BigNumberish]): string
    }>

    transfer: TypedFunctionDescription<{
      encode([recipient, amount]: [string, BigNumberish]): string
    }>

    allowance: TypedFunctionDescription<{
      encode([owner, spender]: [string, string]): string
    }>

    adminMint: TypedFunctionDescription<{
      encode([account, amount]: [string, BigNumberish]): string
    }>

    withdraw: TypedFunctionDescription<{
      encode([account, amount]: [string, BigNumberish]): string
    }>
  }

  events: {
    Transfer: TypedEventDescription<{
      encodeTopics([from, to, value]: [
        string | null,
        string | null,
        null
      ]): string[]
    }>

    Approval: TypedEventDescription<{
      encodeTopics([owner, spender, value]: [
        string | null,
        string | null,
        null
      ]): string[]
    }>
  }
}

export class ArbERC20 extends Contract {
  connect(signerOrProvider: Signer | Provider | string): ArbERC20
  attach(addressOrName: string): ArbERC20
  deployed(): Promise<ArbERC20>

  on(event: EventFilter | string, listener: Listener): ArbERC20
  once(event: EventFilter | string, listener: Listener): ArbERC20
  addListener(eventName: EventFilter | string, listener: Listener): ArbERC20
  removeAllListeners(eventName: EventFilter | string): ArbERC20
  removeListener(eventName: any, listener: Listener): ArbERC20

  interface: ArbERC20Interface

  functions: {
    approve(
      spender: string,
      amount: BigNumberish,
      overrides?: TransactionOverrides
    ): Promise<ContractTransaction>

    totalSupply(): Promise<BigNumber>

    transferFrom(
      sender: string,
      recipient: string,
      amount: BigNumberish,
      overrides?: TransactionOverrides
    ): Promise<ContractTransaction>

    increaseAllowance(
      spender: string,
      addedValue: BigNumberish,
      overrides?: TransactionOverrides
    ): Promise<ContractTransaction>

    balanceOf(account: string): Promise<BigNumber>

    decreaseAllowance(
      spender: string,
      subtractedValue: BigNumberish,
      overrides?: TransactionOverrides
    ): Promise<ContractTransaction>

    transfer(
      recipient: string,
      amount: BigNumberish,
      overrides?: TransactionOverrides
    ): Promise<ContractTransaction>

    allowance(owner: string, spender: string): Promise<BigNumber>

    adminMint(
      account: string,
      amount: BigNumberish,
      overrides?: TransactionOverrides
    ): Promise<ContractTransaction>

    withdraw(
      account: string,
      amount: BigNumberish,
      overrides?: TransactionOverrides
    ): Promise<ContractTransaction>
  }

  approve(
    spender: string,
    amount: BigNumberish,
    overrides?: TransactionOverrides
  ): Promise<ContractTransaction>

  totalSupply(): Promise<BigNumber>

  transferFrom(
    sender: string,
    recipient: string,
    amount: BigNumberish,
    overrides?: TransactionOverrides
  ): Promise<ContractTransaction>

  increaseAllowance(
    spender: string,
    addedValue: BigNumberish,
    overrides?: TransactionOverrides
  ): Promise<ContractTransaction>

  balanceOf(account: string): Promise<BigNumber>

  decreaseAllowance(
    spender: string,
    subtractedValue: BigNumberish,
    overrides?: TransactionOverrides
  ): Promise<ContractTransaction>

  transfer(
    recipient: string,
    amount: BigNumberish,
    overrides?: TransactionOverrides
  ): Promise<ContractTransaction>

  allowance(owner: string, spender: string): Promise<BigNumber>

  adminMint(
    account: string,
    amount: BigNumberish,
    overrides?: TransactionOverrides
  ): Promise<ContractTransaction>

  withdraw(
    account: string,
    amount: BigNumberish,
    overrides?: TransactionOverrides
  ): Promise<ContractTransaction>

  filters: {
    Transfer(from: string | null, to: string | null, value: null): EventFilter

    Approval(
      owner: string | null,
      spender: string | null,
      value: null
    ): EventFilter
  }

  estimate: {
    approve(spender: string, amount: BigNumberish): Promise<BigNumber>

    totalSupply(): Promise<BigNumber>

    transferFrom(
      sender: string,
      recipient: string,
      amount: BigNumberish
    ): Promise<BigNumber>

    increaseAllowance(
      spender: string,
      addedValue: BigNumberish
    ): Promise<BigNumber>

    balanceOf(account: string): Promise<BigNumber>

    decreaseAllowance(
      spender: string,
      subtractedValue: BigNumberish
    ): Promise<BigNumber>

    transfer(recipient: string, amount: BigNumberish): Promise<BigNumber>

    allowance(owner: string, spender: string): Promise<BigNumber>

    adminMint(account: string, amount: BigNumberish): Promise<BigNumber>

    withdraw(account: string, amount: BigNumberish): Promise<BigNumber>
  }
}
