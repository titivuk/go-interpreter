import fs from 'node:fs';

export class BaseClass {

}

export interface Empty {

}

export class Cache extends BaseClass implements Empty {
    private readonly field1: string = "string";

    constructor(param1: number) {
        super()
    }

    public someMethod(): Promise<void> {
        return Promise.resolve()
    }

    public async anotherMethod(param1) {
        let a = 5;

        console.log(a, param1)
    }
}

function fn(param1: any) {
    const ten = 10
}

type User = {
    id: string;
    name: string;
}

interface IUser {
    id: string;
    name: string;
}

const PI = true


fn(false)

enum QWEQWE {

}

1 + 1 
