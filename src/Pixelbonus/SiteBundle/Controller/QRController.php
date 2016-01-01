<?php

namespace Pixelbonus\SiteBundle\Controller;
use Symfony\Bundle\FrameworkBundle\Controller\Controller;
use Sensio\Bundle\FrameworkExtraBundle\Configuration\Route;
use JMS\SecurityExtraBundle\Annotation\Secure;
use Symfony\Component\HttpFoundation\Response;
use Symfony\Component\HttpFoundation\RedirectResponse;
use Symfony\Component\HttpFoundation\StreamedResponse;

use Pixelbonus\SiteBundle\Entity\Course;
use Pixelbonus\SiteBundle\Form\Type\CourseType;

use Pixelbonus\SiteBundle\Entity\QrSet;
use Pixelbonus\SiteBundle\Entity\QrCode;
use Pixelbonus\SiteBundle\Entity\Tag;
use Pixelbonus\SiteBundle\Form\Type\QrSetType;

class QRController extends Controller {
    /**
     * @Route("/courses/{course}", name="course")
     * @Secure(roles="ROLE_USER")
     */
    public function course(Course $course) {
        $user = $this->container->get('security.context')->getToken()->getUser();
        if($course->getUser() != $user) { throw new \Exception('User not authorized to access this course'); }
        if($course->getQrSets()->count() > 0) {
            return $this->render('PixelbonusSiteBundle:QR:course.html.twig', array(
                'course' => $course,
                'qrsets' => $course->getQrSets(),
            ));
        } else {
            $qrset = new QrSet();
            $qrset->setCourse($course);
            $form = $this->createForm(new QrSetType(), $qrset,  array());
            return $this->render('PixelbonusSiteBundle:QR:no_qr_sets.html.twig', array(
                'course' => $course,
                'form' => $form->createView(),
            ));
        }
    }

    /**
     * @Route("/course/{course}/qrset/grades", name="course_grades")
     * @Secure(roles="ROLE_USER")
     */
    public function courseGrades(Course $course) {
        $selectedGradingModel = $this->getRequest()->get('model', 'reduction');
        $redemptions = $this->container->get('doctrine')->getManager()->createQuery('SELECT r.participantNumber, COUNT(r) rcount FROM Pixelbonus\SiteBundle\Entity\Redemption r JOIN r.qrcode qrc JOIN qrc.qrset qr WHERE qr.course = :course GROUP BY r.participantNumber')->setParameter('course', $course)->getResult();
        $maxRedemptions = max(array_map(function($e) {return (int)$e['rcount'];}, $redemptions));
        // Add the grade based on our model
        if($selectedGradingModel == 'reduction') {
            $redemptions = array_map(function($e) use ($maxRedemptions) {
                $e['grade'] = min($e['rcount']/$maxRedemptions*10, 10);
                $e['grade'] = round($e['grade'], 2);
                return $e;
            }, $redemptions);
        } else {
            return new Response('Invalid grading model selected');
        }
        // Sort the grades by the selected field
        $selectedSortField = $this->getRequest()->get('sortBy', 'participantNumber');
        $headerFields = array('participantNumber', 'rcount', 'grade',);
        if(!in_array($selectedSortField, $headerFields)) {
            return new Response('Invalid sort attribute selected');
        }
        uasort($redemptions, function($a, $b) use($selectedSortField) {
            if($a[$selectedSortField] == $b[$selectedSortField]) { return 0; }
            if($a[$selectedSortField] > $b[$selectedSortField]) { return 1; } else { return -1; }
        });
        // Export or render HTML
        if($this->getRequest()->get('export') === 'true') {
            $response = new StreamedResponse(function() use(&$redemptions, &$headerFields) {
                $handle = fopen('php://output', 'r+');

                fputcsv($handle, $headerFields);
                foreach($redemptions as $curRedemption) {
                    fputcsv($handle, $curRedemption);
                }

                fclose($handle);
            });
            $response->headers->set('Content-Type', 'application/force-download');
            $response->headers->set('Content-Disposition','attachment; filename="grades-export.csv"');
            return $response;
        } else {
            return $this->render('PixelbonusSiteBundle:QR:course_grades.html.twig', array(
                'course' => $course,
                'redemptions' => $redemptions,
                'selectedGradingModel' => $selectedGradingModel,
                'selectedSortField' => $selectedSortField,
            ));
        }
    }

    /**
     * @Route("/qrset_new/{course}", name="new_qrset")
     * @Secure(roles="ROLE_USER")
     */
    public function newQrSetAction(Course $course) {
        $user = $this->container->get('security.context')->getToken()->getUser();
        if($course->getUser() != $user) { throw new \Exception('User not authorized to access this course'); }
        $qrset = new QrSet();
        $qrset->setCourse($course);
        $qrset->setQuantity(QrSet::DEFAULT_QUANTITY);
        $form = $this->createForm(new QrSetType(), $qrset,  array());
        if ('POST' == $this->getRequest()->getMethod()) {
            // parameter handling
            $form->bind($this->getRequest());

            if(!$this->getRequest()->request->has($form->getName())) {
                echo 'No form fields specified'; die();
            } else if($form->isValid()) {
                // Set the tags
                $qrset->getTags()->clear();
                foreach($qrset->tagsFromString as $curTag) {
                    $curTagEntity = $this->container->get('doctrine')->getManager()->getRepository('Pixelbonus\SiteBundle\Entity\Tag')->findOneBy(array(
                        'name' => $curTag,
                    ));
                    if(!isset($curTagEntity)) {
                        $curTagEntity = new Tag();
                        $curTagEntity->setName($curTag);
                        $this->container->get('doctrine')->getManager()->persist($curTagEntity);
                        $this->container->get('doctrine')->getManager()->flush($curTagEntity);
                    }
                    $qrset->getTags()->add($curTagEntity);
                }

                $this->container->get('doctrine')->getManager()->persist($qrset);
                $this->container->get('doctrine')->getManager()->flush($qrset);

                return new RedirectResponse($this->container->get('router')->generate('download_qr', array('qrset' => $qrset->getId(), 'quantity' => $qrset->getQuantity())));
            }
        }
        // Get existing tags
        $existingTags = array();
        foreach($course->getQrSets() as $curQrSet) {
            foreach($curQrSet->getTags() as $curTag) {
                $existingTags[] = $curTag;
            }
        }
        return $this->render('PixelbonusSiteBundle:QR:new.html.twig', array(
            'course' => $course,
            'form' => $form->createView(),
            'existingTags' => $existingTags,
        ));
    }

    /**
     * @Route("/qrset/{qrset}", name="qrset")
     * @Secure(roles="ROLE_USER")
     */
    public function qrset(QrSet $qrset) {
        return $this->render('PixelbonusSiteBundle:QR:qr_set.html.twig', array(
            'qrset' => $qrset,
        ));
    }

    /**
     * @Route("/qrset/{qrset}/download", name="download_qr")
     * @Secure(roles="ROLE_USER")
     */
    public function downloadQr(QrSet $qrset) {
        if($this->getRequest()->get('quantity') == null) { echo 'Quantity is required'; die(); }
        return $this->render('PixelbonusSiteBundle:QR:download.html.twig', array(
            'qrset' => $qrset,
            'quantity' => $this->getRequest()->get('quantity'),
        ));
    }

    /**
     * @Route("/qrset/{qrset}/generate", name="generate_qr")
     * @Secure(roles="ROLE_USER")
     */
    public function generateQr(QrSet $qrset) {
        if($this->getRequest()->get('quantity') == null) { echo 'Quantity is required'; die(); }
        $qrImages = array();
        $toFlush = array($qrset);
        for($i = $qrset->getQrcodes()->count(); $i < $qrset->getQrcodes()->count()+(int)$this->getRequest()->get('quantity'); $i++) {
            // Create the QR Entity
            $qrCode = new QrCode();
            $qrCode->setQrset($qrset);
            $hash = hash_hmac('sha1', $qrset->getId().'_'.$i, $this->container->getParameter("secret"));
            $qrCode->setCode($hash);
            $this->container->get('doctrine')->getManager()->persist($qrCode);
            $toFlush[] = $qrCode;

            // Generate QR image based on the created entity
            $qrImage = array();
            $fileName = tempnam($this->container->getParameter("kernel.cache_dir"), 'qrimg');
            $link = $this->container->get('router')->generate('redeem', array('hash' => $qrCode->getCode()), true);
            \QRcode::svg($link, $fileName);
            $qrImage['link'] = $qrCode->getCode();
            $qrImage['svg'] = file_get_contents($fileName);
            $qrImages[] = $qrImage;
        }
        $this->container->get('doctrine')->getManager()->persist($qrset);
        $this->container->get('doctrine')->getManager()->flush($toFlush);
        $html = $this->container->get('templating')->render('PixelbonusSiteBundle:QR:qr_template.html.twig', array(
            'qrImages' => $qrImages,
        ));
        if($this->getRequest()->get('html') == 'true') {
            return new Response($html);
        } else {
            $pdf = $this->container->get('knp_snappy.pdf')->getOutputFromHtml($html);
            return new Response($pdf, 200, array(
                'Content-Type' => 'application/pdf',
                'Content-Disposition' => sprintf('attachment;filename="%s.pdf"', 'qr'),
            ));
        }
    }

    /**
     * @Route("/qrset/{qrset}/delete", name="delete_qrset")
     * @Secure(roles="ROLE_USER")
     */
    public function deleteQr(QrSet $qrset) {
        $course = $qrset->getCourse()->getId();
        $this->container->get('doctrine')->getManager()->remove($qrset);
        $this->container->get('doctrine')->getManager()->flush($qrset);
        return new RedirectResponse($this->container->get('router')->generate('course', array('course' => $course)));
    }
}
