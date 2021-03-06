package logic

import (
    "os"
    "log"
    "sona/protocol"
    "sona/broker/conf"
    "sona/common/net/tcp"
    "github.com/golang/protobuf/proto"
)

//全局：broker server，服务于agent
var BrokerServer *tcp.Server

//消息ID与对应PB的映射
func brokerMsgFactory(cmdId uint) proto.Message {
    switch cmdId {
    case protocol.SubscribeReqId:
        return &protocol.SubscribeReq{}
    case protocol.PullServiceConfigReqId:
        return &protocol.PullServiceConfigReq{}
    }
    return nil
}

//SubscribeReqId消息的回调函数
func SubscribeHandler(session *tcp.Session, pb proto.Message) {
    req := pb.(*protocol.SubscribeReq)
    log.Printf("agent %s tries to subscribe service %s\n", session.Addr(), *req.ServiceKey)
    //订阅：此连接对*req.ServiceKey感兴趣
    session.Subscribe(*req.ServiceKey)
    //创建回包
    rsp := protocol.SubscribeBrokerRsp{}
    rsp.ServiceKey = proto.String(*req.ServiceKey)
    //查看是否有此配置
    keys, values, version := CacheLayer.GetData(*req.ServiceKey)
    if version == 0 {
        log.Printf("subscribe service %s failed because of no this service\n", *req.ServiceKey)
        rsp.Code = proto.Int32(-1)//订阅失败
        rsp.Error = proto.String("data does not exist")
        rsp.Version = proto.Uint32(0)
    } else {
        if len(keys) == 0 {
            log.Printf("subscribe service %s failed because of this service's data is empty\n", *req.ServiceKey)
            rsp.Code = proto.Int32(-1)//订阅失败
            rsp.Error = proto.String("data is empty")
            rsp.Version = proto.Uint32(0)
        } else {
            log.Printf("subscribe service %s successfully\n", *req.ServiceKey)
            rsp.Code = proto.Int32(0)//订阅成功
            //填充配置
            rsp.Version = proto.Uint32(uint32(version))
            rsp.ConfKeys = keys
            rsp.Values = values
        }
    }
    //回包
    session.SendData(protocol.SubscribeBrokerRspId, &rsp)
}

//PullServiceConfigReqId消息的回调函数
func PullConfigHandler(session *tcp.Session, pb proto.Message) {
    log.Println("DEBUG: into pull configure request callback")
    req := pb.(*protocol.PullServiceConfigReq)
    //订阅：此连接对*req.ServiceKey感兴趣
    session.Subscribe(*req.ServiceKey)
    //创建回包
    rsp := protocol.PullServiceConfigRsp{}
    rsp.ServiceKey = proto.String(*req.ServiceKey)

    //查看是否有此配置
    keys, values, version := CacheLayer.GetData(*req.ServiceKey)
    rsp.Version = proto.Uint32(uint32(version))
    rsp.ConfKeys = keys
    rsp.Values = values

    //回包
    session.SendData(protocol.PullServiceConfigRspId, &rsp)
}

func StartBrokerService() {
    server, err := tcp.CreateServer("broker", "0.0.0.0", conf.GlobalConf.BrokerPort, uint32(conf.GlobalConf.BrokerConnectionLimit))
    if err != nil {
        log.Println(err)
        os.Exit(1)
    }
    BrokerServer = server
    //开启心跳检测
    BrokerServer.EnableHeartbeat()
    //注册消息ID与PB的映射
    BrokerServer.SetFactory(brokerMsgFactory)
    //注册所有回调
    BrokerServer.RegHandler(protocol.SubscribeReqId, SubscribeHandler)
    BrokerServer.RegHandler(protocol.PullServiceConfigReqId, PullConfigHandler)
    //启动服务
    BrokerServer.Start()
}
