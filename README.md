# Jeongseup Chain

### Tests

```bash
# unit testing for genaccount
gotest ./cmd/jeongseupd/cmd -v
```

### References

1. https://github.com/cosmos/cosmos-sdk/blob/v0.45.4/simapp/app.go#L140
2. https://github.com/xpladev/xpla
3. https://github.com/cosmosregistry/chain-minimal.git
4. https://ida.interchain.io

#### Memo..

1. staking 모듈은 반드시 필요함 gentxs 커맨드 호출 시 필요..
   https://github.com/cosmos/cosmos-sdk/blob/v0.45.4/x/genutil/gentx.go#L46C24-L46C39
2. `moduleAccountPermissions = map[string][]string{}` 도 start cmd시 해당 모듈 어카운트 주소를 통해서 validate하는 로직이 있어서 필요한 듯?
   https://github.com/cosmos/cosmos-sdk/blob/v0.45.4/x/staking/keeper/keeper.go#L45C1-L47C3
3. 그리고 나서 또 컨센서스에 막힘..
   ..
