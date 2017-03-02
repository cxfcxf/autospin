package main

import (
	"fmt"
	"sync"
	"flag"
	"context"
	"golang.org/x/oauth2"
	"github.com/digitalocean/godo"
)

var (
	c = flag.Bool("c",false,"create droplets with imageid and location")
	d = flag.Bool("d",false,"delete all droplets")
	l = flag.Bool("l",false,"list proxy")
	p = flag.String("p","","api token")
	n = flag.Int("n",1,"number of droplets to spin up")
	id = flag.Int("id",1,"snapshot image id")
	lo = flag.String("lo","sfo1","launch location")
	pre = flag.String("pre","proxy-","prefix of the instance name")
)

type TokenSource struct {
	AccessToken string
}

func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

func setRequest(dropletName, region string, imageid int, droplets chan *godo.DropletCreateRequest) {
	sshkey := godo.DropletCreateSSHKey{
		ID: 3456386,
	}
	droplets <- &godo.DropletCreateRequest{
		Name: dropletName,
		Region: region,
		Size: "512mb",
		Image: godo.DropletCreateImage{
			ID: imageid,
		},
		SSHKeys: []godo.DropletCreateSSHKey{sshkey},
	}
}

func createDroplet(client *godo.Client, droplet *godo.DropletCreateRequest, wg *sync.WaitGroup) {
	defer wg.Done()
	_, _, err := client.Droplets.Create(context.TODO(), droplet)
	if err != nil {
		fmt.Println("failed to create droplet: ", droplet.Name, err)
	}
}

func getDroplet(client *godo.Client) ([]int, []string) {
	var dropletids []int
	var dropletips []string
	listoptions := &godo.ListOptions{}
	droplet, _, _ := client.Droplets.List(context.TODO(), listoptions)
	for _, d := range droplet {
		ip, _ := d.PublicIPv4()
		dropletips = append(dropletips, ip)
		dropletids = append(dropletids, d.ID)
	}
	return dropletids, dropletips
}

func DeleteDroplet(client *godo.Client, dropletid int, wg *sync.WaitGroup) {
	defer wg.Done()
	_, err := client.Droplets.Delete(context.TODO(), dropletid)
	if err != nil {
		fmt.Println("failed to delete droplet: ", dropletid, err)
	}
}

func main() {
	flag.Parse()

	tokenSource := &TokenSource {
		AccessToken: *p,
	}

	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	droplets := make(chan *godo.DropletCreateRequest)

	if *c {
		for i := 1; i <= *n; i++ {
			dropletName := fmt.Sprintf("%s%d", *pre, i)
			go setRequest(dropletName, *lo, *id, droplets)
		}
		var wg sync.WaitGroup

		for j := 1; j <= *n; j++ {
			droplet := <-droplets
			wg.Add(1)
			go createDroplet(client, droplet, &wg)
		}
		wg.Wait()

		close(droplets)
	}

	if *d {
		var wg sync.WaitGroup
		dropletids, _ := getDroplet(client)
		for _, dropletid := range dropletids {
			wg.Add(1)
			go DeleteDroplet(client, dropletid, &wg)
		}
		wg.Wait()
	}

	if *l {
		_, dropletips := getDroplet(client)
		for _, dropletip := range dropletips {
			fmt.Printf("%s:port:user:pass\n",dropletip)
		}
	}
}
